use std::sync::Arc;
use anyhow::Result;
use geo::{Point, LineString};
use tracing::{debug, info, warn};
use uuid::Uuid;
use chrono::Utc;

use crate::models::*;
use crate::osrm_client::OsrmClient;
use crate::corridor_manager::CorridorManager;
use crate::error::RoutingError;

pub struct PriorityRouter {
    osrm: Arc<OsrmClient>,
    corridor_manager: Arc<CorridorManager>,
}

impl PriorityRouter {
    pub fn new(osrm: Arc<OsrmClient>, corridor_manager: Arc<CorridorManager>) -> Self {
        Self { osrm, corridor_manager }
    }

    pub async fn calculate_route(&self, request: RouteRequest) -> Result<RouteResponse> {
        let start = std::time::Instant::now();

        // Validate priority level
        if request.priority_level > 3 {
            return Err(RoutingError::UnsupportedPriority(request.priority_level).into());
        }

        // Determine OSRM profile based on priority and vehicle type
        let profile = self.select_profile(&request);

        // Check if we need to avoid reserved lanes for public traffic
        let avoid_reserved = request.priority_level == 0;

        // Calculate primary route
        let primary = self.route_with_profile(&request, &profile, avoid_reserved).await?;

        // Calculate alternatives for public traffic
        let alternative_routes = if request.return_alternatives && request.priority_level == 0 {
            self.calculate_alternatives(&request, &profile, &primary, avoid_reserved).await?
        } else {
            vec![]
        };

        let elapsed = start.elapsed();
        debug!("Route calculated in {:.2}ms", elapsed.as_millis());

        Ok(RouteResponse {
            request_id: request.request_id,
            primary_route: primary,
            alternative_routes,
            status: RouteStatus::Ok,
            error_message: None,
            calculated_at: Utc::now(),
        })
    }

    fn select_profile(&self, request: &RouteRequest) -> String {
        match (request.priority_level, request.vehicle_type.as_deref()) {
            (2, Some("ambulance")) | (2, Some("fire")) | (2, Some("police")) => "emergency".to_string(),
            (3, _) => "emergency".to_string(), // VIP gets emergency routing
            _ => match request.profile {
                RoutingProfile::Car => "car",
                RoutingProfile::Bike => "bike",
                RoutingProfile::Foot => "foot",
                RoutingProfile::Emergency => "emergency",
            }.to_string(),
        }
    }

    async fn route_with_profile(
        &self,
        request: &RouteRequest,
        profile: &str,
        avoid_reserved: bool,
    ) -> Result<Route> {
        // Get route from OSRM
        let osrm_resp = self.osrm
            .route(request.origin, request.destination, profile, false, true)
            .await?;

        let osrm_route = &osrm_resp.routes[0];

        // Convert geometry
        let geometry = self.osrm.convert_geometry(&osrm_route.geometry);

        // Extract segment info
        let mut segments = self.osrm.extract_segments(osrm_route);

        // Enrich with real-time data (congestion, reservations)
        self.enrich_segments(&mut segments, avoid_reserved).await;

        // Calculate congestion level
        let congestion_level = self.calculate_route_congestion(&segments);

        // Check for reserved lane usage
        let uses_reserved = segments.iter().any(|s| s.is_reserved);

        // Check for incidents on route
        let warnings = self.check_incidents_on_route(&geometry).await;

        Ok(Route {
            route_id: Uuid::new_v4().to_string(),
            geometry,
            segments,
            distance_meters: osrm_route.distance as i64,
            duration_seconds: osrm_route.duration as i32,
            duration_with_traffic_seconds: self.calculate_traffic_duration(&segments),
            congestion_level,
            uses_reserved_lanes: uses_reserved,
            warnings,
            metadata: serde_json::json!({
                "profile": profile,
                "priority_level": request.priority_level,
            }),
        })
    }

    async fn enrich_segments(&self, segments: &mut [SegmentInfo], avoid_reserved: bool) {
        // In production, this would query:
        // 1. Traffic speeds from Redis/PostgreSQL
        // 2. Active corridor reservations
        // 3. Incident data
        // For now, simulate with mock data

        for segment in segments.iter_mut() {
            // Simulate traffic data
            segment.current_speed = (segment.max_speed as f32 * 0.7) as i32;
            segment.congestion_level = match segment.current_speed as f32 / segment.max_speed as f32 {
                s if s > 0.8 => 0,
                s if s > 0.6 => 1,
                s if s > 0.4 => 2,
                s if s > 0.2 => 3,
                s if s > 0.1 => 4,
                _ => 5,
            };

            // Check reservations
            segment.is_reserved = self.corridor_manager.is_segment_reserved(segment.segment_id).await;

            // If public traffic and segment is reserved, mark as avoided
            if avoid_reserved && segment.is_reserved {
                segment.congestion_level = 5; // Effectively blocked
            }
        }
    }

    fn calculate_route_congestion(&self, segments: &[SegmentInfo]) -> i32 {
        segments.iter()
            .map(|s| s.congestion_level)
            .max()
            .unwrap_or(0)
    }

    fn calculate_traffic_duration(&self, segments: &[SegmentInfo]) -> i32 {
        segments.iter()
            .map(|s| {
                let speed = s.current_speed.max(1) as f64;
                (s.length_meters as f64 / (speed / 3.6)).ceil() as i32
            })
            .sum()
    }

    async fn check_incidents_on_route(&self, geometry: &LineString<f64>) -> Vec<IncidentWarning> {
        // In production, query incidents that intersect route buffer
        // For now, return empty
        vec![]
    }

    async fn calculate_alternatives(
        &self,
        request: &RouteRequest,
        profile: &str,
        primary: &Route,
        avoid_reserved: bool,
    ) -> Result<Vec<Route>> {
        let mut alternatives = Vec::new();

        // Strategy 1: Slightly different origin/destination (perturbation)
        // Strategy 2: Avoid primary route's major segments
        // Strategy 3: Different profile (e.g., avoid highways)

        // For now, generate mock alternatives with different congestion
        for i in 0..request.max_alternatives.min(3) {
            let mut alt = primary.clone();
            alt.route_id = Uuid::new_v4().to_string();
            alt.distance_meters = (primary.distance_meters as f64 * (1.0 + (i as f64 + 1.0) * 0.1)) as i64;
            alt.duration_with_traffic_seconds = (primary.duration_with_traffic_seconds as f64 * (1.0 + (i as f64 + 1.0) * 0.15)) as i32;
            alt.congestion_level = (primary.congestion_level + i as i32 + 1).min(5);
            alt.metadata = serde_json::json!({
                "alternative_index": i + 1,
                "profile": profile,
            });
            alternatives.push(alt);
        }

        Ok(alternatives)
    }

    pub async fn stream_route_updates(&self, request: RouteRequest) -> Result<tokio::sync::mpsc::Receiver<RouteUpdate>> {
        let (tx, rx) = tokio::sync::mpsc::channel(100);

        // In production, this would:
        // 1. Subscribe to Kafka topics for traffic/incidents
        // 2. Monitor corridor reservations
        // 3. Push updates when route conditions change

        tokio::spawn(async move {
            let mut interval = tokio::time::interval(std::time::Duration::from_secs(30));
            loop {
                interval.tick().await;
                // Send periodic updates
                if tx.send(RouteUpdate {
                    request_id: request.request_id.clone(),
                    update_type: RouteUpdateType::Congestion,
                    updated_route: None,
                    new_incident: None,
                    congestion_change: None,
                    timestamp: Utc::now().timestamp_millis(),
                }).await.is_err() {
                    break;
                }
            }
        });

        Ok(rx)
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RouteUpdate {
    pub request_id: String,
    pub update_type: RouteUpdateType,
    pub updated_route: Option<Route>,
    pub new_incident: Option<IncidentWarning>,
    pub congestion_change: Option<CongestionChange>,
    pub timestamp: i64,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq)]
pub enum RouteUpdateType {
    Recalculated,
    Incident,
    Congestion,
    Arrived,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CongestionChange {
    pub segment_id: i64,
    pub old_level: i32,
    pub new_level: i32,
}