use serde::{Deserialize, Serialize};
use geo::{Point, LineString};
use uuid::Uuid;
use chrono::{DateTime, Utc};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RouteRequest {
    pub request_id: String,
    pub priority_level: i32,
    pub origin: Point,
    pub destination: Point,
    pub vehicle_id: Option<String>,
    pub vehicle_type: Option<String>,
    pub avoid_segments: Vec<i64>,
    pub return_alternatives: bool,
    pub max_alternatives: usize,
    pub profile: RoutingProfile,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq)]
pub enum RoutingProfile {
    Car,
    Bike,
    Foot,
    Emergency,
}

impl Default for RoutingProfile {
    fn default() -> Self {
        RoutingProfile::Car
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RouteResponse {
    pub request_id: String,
    pub primary_route: Route,
    pub alternative_routes: Vec<Route>,
    pub status: RouteStatus,
    pub error_message: Option<String>,
    pub calculated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq)]
pub enum RouteStatus {
    Ok,
    NoRoute,
    Partial,
    Error,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Route {
    pub route_id: String,
    pub geometry: LineString<f64>,
    pub segments: Vec<SegmentInfo>,
    pub distance_meters: i64,
    pub duration_seconds: i32,
    pub duration_with_traffic_seconds: i32,
    pub congestion_level: i32,
    pub uses_reserved_lanes: bool,
    pub warnings: Vec<IncidentWarning>,
    pub metadata: serde_json::Value,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SegmentInfo {
    pub segment_id: i64,
    pub osm_id: i64,
    pub name: Option<String>,
    pub highway_type: String,
    pub max_speed: i32,
    pub current_speed: i32,
    pub congestion_level: i32,
    pub length_meters: i64,
    pub duration_seconds: i32,
    pub is_reserved: bool,
    pub is_oneway: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IncidentWarning {
    pub incident_id: String,
    pub location: Point,
    pub incident_type: String,
    pub severity: i32,
    pub description: String,
    pub delay_seconds: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CorridorRequest {
    pub reservation_id: String,
    pub vehicle_id: String,
    pub corridor_path: LineString<f64>,
    pub lookahead_meters: u32,
    pub expires_at: DateTime<Utc>,
    pub priority_level: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CorridorResponse {
    pub reservation_id: String,
    pub success: bool,
    pub error_message: Option<String>,
    pub reserved_segment_ids: Vec<i64>,
    pub expires_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CorridorReservation {
    pub reservation_id: String,
    pub vehicle_id: String,
    pub segment_ids: Vec<i64>,
    pub expires_at: DateTime<Utc>,
    pub priority_level: i32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CongestionSegment {
    pub segment_id: i64,
    pub osm_id: i64,
    pub geometry: LineString<f64>,
    pub congestion_level: i32,
    pub current_speed: i32,
    pub max_speed: i32,
}

impl Default for RouteRequest {
    fn default() -> Self {
        Self {
            request_id: Uuid::new_v4().to_string(),
            priority_level: 0,
            origin: Point::new(0.0, 0.0),
            destination: Point::new(0.0, 0.0),
            vehicle_id: None,
            vehicle_type: None,
            avoid_segments: vec![],
            return_alternatives: true,
            max_alternatives: 3,
            profile: RoutingProfile::Car,
        }
    }
}