use tonic::{Request, Response, Status, Streaming};
use std::sync::Arc;
use futures::stream;
use tokio_stream::wrappers::ReceiverStream;
use uuid::Uuid;
use chrono::Utc;
use tracing::{debug, info, warn};

use crate::models::*;
use crate::priority_router::PriorityRouter;
use crate::corridor_manager::CorridorManager;
use crate::error::RoutingError;

// Generated from proto
pub mod routing {
    tonic::include_proto!("aegisroad.routing");
}

use routing::{
    routing_service_server::{RoutingService, RoutingServiceServer},
    *,
};

pub struct RoutingServiceImpl {
    router: Arc<PriorityRouter>,
    corridor_manager: Arc<CorridorManager>,
}

impl RoutingServiceImpl {
    pub fn new(router: Arc<PriorityRouter>, corridor_manager: Arc<CorridorManager>) -> Self {
        Self { router, corridor_manager }
    }

    pub fn into_router(self) -> RoutingServiceServer<Self> {
        RoutingServiceServer::new(self)
    }
}

#[tonic::async_trait]
impl RoutingService for RoutingServiceImpl {
    async fn calculate_route(
        &self,
        request: Request<RouteRequest>,
    ) -> Result<Response<RouteResponse>, Status> {
        let req = request.into_inner();
        debug!("CalculateRoute request: {}", req.request_id);

        let route_req = convert_request(req)?;
        let response = self.router.calculate_route(route_req).await
            .map_err(|e| Status::internal(e.to_string()))?;

        Ok(Response::new(convert_response(response)))
    }

    type StreamRouteUpdatesStream = ReceiverStream<Result<RouteUpdate, Status>>;

    async fn stream_route_updates(
        &self,
        request: Request<RouteRequest>,
    ) -> Result<Response<Self::StreamRouteUpdatesStream>, Status> {
        let req = request.into_inner();
        debug!("StreamRouteUpdates request: {}", req.request_id);

        let route_req = convert_request(req)?;
        let rx = self.router.stream_route_updates(route_req).await
            .map_err(|e| Status::internal(e.to_string()))?;

        let stream = ReceiverStream::new(rx).map(|update| {
            update.map(convert_route_update).map_err(|e| Status::internal(e.to_string()))
        });

        Ok(Response::new(stream))
    }

    async fn reserve_corridor(
        &self,
        request: Request<CorridorRequest>,
    ) -> Result<Response<CorridorResponse>, Status> {
        let req = request.into_inner();
        debug!("ReserveCorridor request: {}", req.reservation_id);

        let corridor_req = convert_corridor_request(req)?;
        let response = self.corridor_manager.reserve(corridor_req).await
            .map_err(|e| Status::internal(e.to_string()))?;

        Ok(Response::new(convert_corridor_response(response)))
    }

    async fn release_corridor(
        &self,
        request: Request<CorridorRelease>,
    ) -> Result<Response<()>, Status> {
        let req = request.into_inner();
        debug!("ReleaseCorridor request: {}", req.reservation_id);

        self.corridor_manager.release(&req.reservation_id).await
            .map_err(|e| Status::internal(e.to_string()))?;

        Ok(Response::new(()))
    }

    async fn get_congestion_map(
        &self,
        request: Request<CongestionRequest>,
    ) -> Result<Response<CongestionResponse>, Status> {
        let req = request.into_inner();
        debug!("GetCongestionMap request");

        // In production, query PostGIS for congestion data in bbox
        // For now, return mock data
        let response = CongestionResponse {
            segments: vec![],
            timestamp: Utc::now().timestamp_millis(),
        };

        Ok(Response::new(response))
    }
}

// Conversion functions between proto and internal types

fn convert_request(proto_req: RouteRequest) -> Result<crate::models::RouteRequest, Status> {
    let origin = Point::new(proto_req.origin.longitude, proto_req.origin.latitude);
    let destination = Point::new(proto_req.destination.longitude, proto_req.destination.latitude);

    let profile = match proto_req.profile {
        1 => RoutingProfile::Car,
        2 => RoutingProfile::Bike,
        3 => RoutingProfile::Foot,
        4 => RoutingProfile::Emergency,
        _ => RoutingProfile::Car,
    };

    Ok(crate::models::RouteRequest {
        request_id: proto_req.request_id,
        priority_level: proto_req.priority_level,
        origin,
        destination,
        vehicle_id: if proto_req.vehicle_id.is_empty() { None } else { Some(proto_req.vehicle_id) },
        vehicle_type: if proto_req.vehicle_type.is_empty() { None } else { Some(proto_req.vehicle_type) },
        avoid_segments: proto_req.avoid_segments,
        return_alternatives: proto_req.return_alternatives,
        max_alternatives: proto_req.max_alternatives as usize,
        profile,
    })
}

fn convert_response(resp: crate::models::RouteResponse) -> RouteResponse {
    RouteResponse {
        request_id: resp.request_id,
        primary_route: Some(convert_route(resp.primary_route)),
        alternative_routes: resp.alternative_routes.into_iter().map(convert_route).collect(),
        status: resp.status as i32,
        error_message: resp.error_message.unwrap_or_default(),
        calculated_at: resp.calculated_at.timestamp_millis(),
    }
}

fn convert_route(route: crate::models::Route) -> Route {
    Route {
        route_id: route.route_id,
        geometry: route.geometry.0.into_iter().map(|p| Point {
            longitude: p.x(),
            latitude: p.y(),
        }).collect(),
        segments: route.segments.into_iter().map(convert_segment).collect(),
        distance_meters: route.distance_meters,
        duration_seconds: route.duration_seconds,
        duration_with_traffic_seconds: route.duration_with_traffic_seconds,
        congestion_level: route.congestion_level,
        uses_reserved_lanes: route.uses_reserved_lanes,
        warnings: route.warnings.into_iter().map(convert_warning).collect(),
        metadata: route.metadata.to_string(),
    }
}

fn convert_segment(seg: crate::models::SegmentInfo) -> SegmentInfo {
    SegmentInfo {
        segment_id: seg.segment_id,
        osm_id: seg.osm_id,
        name: seg.name.unwrap_or_default(),
        highway_type: seg.highway_type,
        max_speed: seg.max_speed,
        current_speed: seg.current_speed,
        congestion_level: seg.congestion_level,
        length_meters: seg.length_meters,
        duration_seconds: seg.duration_seconds,
        is_reserved: seg.is_reserved,
        is_oneway: seg.is_oneway,
    }
}

fn convert_warning(warn: crate::models::IncidentWarning) -> IncidentWarning {
    IncidentWarning {
        incident_id: warn.incident_id,
        location: Some(Point { longitude: warn.location.x(), latitude: warn.location.y() }),
        incident_type: warn.incident_type,
        severity: warn.severity,
        description: warn.description,
        delay_seconds: warn.delay_seconds,
    }
}

fn convert_corridor_request(proto_req: CorridorRequest) -> Result<crate::models::CorridorRequest, Status> {
    let corridor_path = LineString::from(proto_req.corridor_path.into_iter().map(|p| {
        Point::new(p.longitude, p.latitude)
    }).collect::<Vec<_>>());

    Ok(crate::models::CorridorRequest {
        reservation_id: proto_req.reservation_id,
        vehicle_id: proto_req.vehicle_id,
        corridor_path,
        lookahead_meters: proto_req.lookahead_meters,
        expires_at: chrono::DateTime::from_timestamp_millis(proto_req.expires_at).unwrap_or(Utc::now()),
        priority_level: proto_req.priority_level,
    })
}

fn convert_corridor_response(resp: CorridorResponse) -> CorridorResponse {
    resp
}

fn convert_route_update(update: crate::priority_router::RouteUpdate) -> RouteUpdate {
    RouteUpdate {
        request_id: update.request_id,
        r#type: update.update_type as i32,
        updated_route: update.updated_route.map(convert_route),
        new_incident: update.new_incident.map(convert_warning),
        congestion_change: update.congestion_change.map(|c| CongestionChange {
            segment_id: c.segment_id,
            old_level: c.old_level,
            new_level: c.new_level,
        }),
        timestamp: update.timestamp,
    }
}

// Helper to include geo types in proto conversion
use geo::{Point, LineString};