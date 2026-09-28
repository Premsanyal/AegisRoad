use std::sync::Arc;
use std::collections::HashMap;
use std::time::Duration;
use anyhow::Result;
use tokio::sync::RwLock;
use tracing::{debug, info, warn};
use uuid::Uuid;
use chrono::{DateTime, Utc};

use crate::models::{CorridorRequest, CorridorResponse, CorridorReservation};

pub struct CorridorManager {
    reservations: Arc<RwLock<HashMap<String, CorridorReservation>>>,
    segment_reservations: Arc<RwLock<HashMap<i64, Vec<String>>>>, // segment_id -> reservation_ids
}

impl CorridorManager {
    pub fn new() -> Self {
        Self {
            reservations: Arc::new(RwLock::new(HashMap::new())),
            segment_reservations: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    pub async fn reserve(&self, request: CorridorRequest) -> Result<CorridorResponse> {
        let mut reservations = self.reservations.write().await;
        let mut segment_reservations = self.segment_reservations.write().await;

        // Check for conflicts with higher priority reservations
        for segment_id in &request.corridor_path {
            // Simplified: in production, extract segment IDs from path
            if let Some(existing) = segment_reservations.get(segment_id) {
                for res_id in existing {
                    if let Some(res) = reservations.get(res_id) {
                        if res.priority_level > request.priority_level {
                            return Ok(CorridorResponse {
                                reservation_id: request.reservation_id,
                                success: false,
                                error_message: Some(format!("Segment {} reserved by higher priority vehicle", segment_id)),
                                reserved_segment_ids: vec![],
                                expires_at: Utc::now(),
                            });
                        }
                    }
                }
            }
        }

        // Create reservation
        let reservation = CorridorReservation {
            reservation_id: request.reservation_id.clone(),
            vehicle_id: request.vehicle_id.clone(),
            segment_ids: request.corridor_path.clone(), // In production, convert path to segment IDs
            expires_at: request.expires_at,
            priority_level: request.priority_level,
        };

        // Register reservation
        for segment_id in &reservation.segment_ids {
            segment_reservations
                .entry(*segment_id)
                .or_default()
                .push(request.reservation_id.clone());
        }

        reservations.insert(request.reservation_id.clone(), reservation);

        info!("Corridor reserved: {} for vehicle {}", request.reservation_id, request.vehicle_id);

        Ok(CorridorResponse {
            reservation_id: request.reservation_id,
            success: true,
            error_message: None,
            reserved_segment_ids: request.corridor_path,
            expires_at: request.expires_at,
        })
    }

    pub async fn release(&self, reservation_id: &str) -> Result<bool> {
        let mut reservations = self.reservations.write().await;
        let mut segment_reservations = self.segment_reservations.write().await;

        if let Some(reservation) = reservations.remove(reservation_id) {
            for segment_id in &reservation.segment_ids {
                if let Some(list) = segment_reservations.get_mut(segment_id) {
                    list.retain(|id| id != reservation_id);
                    if list.is_empty() {
                        segment_reservations.remove(segment_id);
                    }
                }
            }
            info!("Corridor released: {}", reservation_id);
            Ok(true)
        } else {
            warn!("Attempted to release non-existent reservation: {}", reservation_id);
            Ok(false)
        }
    }

    pub async fn is_segment_reserved(&self, segment_id: i64) -> bool {
        let segment_reservations = self.segment_reservations.read().await;
        segment_reservations.get(&segment_id).map_or(false, |list| !list.is_empty())
    }

    pub async fn get_reservation(&self, reservation_id: &str) -> Option<CorridorReservation> {
        let reservations = self.reservations.read().await;
        reservations.get(reservation_id).cloned()
    }

    pub async fn cleanup_expired(&self) {
        let mut reservations = self.reservations.write().await;
        let mut segment_reservations = self.segment_reservations.write().await;
        let now = Utc::now();

        let expired: Vec<String> = reservations
            .iter()
            .filter(|(_, v)| v.expires_at < now)
            .map(|(k, _)| k.clone())
            .collect();

        for id in expired {
            if let Some(reservation) = reservations.remove(&id) {
                for segment_id in &reservation.segment_ids {
                    if let Some(list) = segment_reservations.get_mut(segment_id) {
                        list.retain(|rid| rid != &id);
                        if list.is_empty() {
                            segment_reservations.remove(segment_id);
                        }
                    }
                }
                debug!("Cleaned up expired reservation: {}", id);
            }
        }
    }

    pub async fn start_cleanup_task(self: Arc<Self>) {
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(Duration::from_secs(60));
            loop {
                interval.tick().await;
                self.cleanup_expired().await;
            }
        });
    }
}

impl Default for CorridorManager {
    fn default() -> Self {
        Self::new()
    }
}