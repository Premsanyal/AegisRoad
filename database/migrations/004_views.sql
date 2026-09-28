-- 004_views.sql
-- Materialized views and helper views for common queries

-- Live congestion view (real-time)
CREATE OR REPLACE VIEW live_congestion AS
SELECT
    r.osm_id as id,
    r.osm_id,
    r.geometry,
    r.highway_type,
    r.name,
    r.max_speed,
    r.lanes,
    r.is_oneway,
    r.priority_tier,
    COALESCE(t.speed_kmh, r.max_speed) as current_speed,
    CASE
        WHEN COALESCE(t.speed_kmh, r.max_speed) / NULLIF(r.max_speed, 0) > 0.8 THEN 0  -- Free flow
        WHEN COALESCE(t.speed_kmh, r.max_speed) / NULLIF(r.max_speed, 0) > 0.6 THEN 1  -- Light
        WHEN COALESCE(t.speed_kmh, r.max_speed) / NULLIF(r.max_speed, 0) > 0.4 THEN 2  -- Moderate
        WHEN COALESCE(t.speed_kmh, r.max_speed) / NULLIF(r.max_speed, 0) > 0.2 THEN 3  -- Heavy
        WHEN COALESCE(t.speed_kmh, r.max_speed) / NULLIF(r.max_speed, 0) > 0.1 THEN 4  -- Severe
        ELSE 5  -- Gridlock
    END as congestion_level,
    t.confidence,
    t.source,
    t.timestamp as speed_timestamp,
    GREATEST(r.updated_at, t.timestamp) as last_updated
FROM road_segments r
LEFT JOIN LATERAL (
    SELECT speed_kmh, confidence, source, timestamp
    FROM traffic_speeds ts
    WHERE ts.segment_id = r.osm_id
    ORDER BY ts.timestamp DESC
    LIMIT 1
) t ON true;

-- Active incidents with details
CREATE OR REPLACE VIEW active_incidents AS
SELECT
    i.id,
    i.type,
    i.severity,
    i.geometry,
    i.affected_segments,
    i.description,
    i.reported_by,
    i.status,
    i.created_at,
    i.assigned_services,
    v.callsign as assigned_callsign,
    v.type as assigned_type,
    v.current_location as vehicle_location,
    EXTRACT(EPOCH FROM (NOW() - i.created_at)) / 60 as minutes_active
FROM incidents i
LEFT JOIN LATERAL (
    SELECT callsign, type, current_location
    FROM vehicles
    WHERE id = ANY(i.assigned_services)
    ORDER BY
        CASE type WHEN 'ambulance' THEN 1 WHEN 'fire' THEN 2 WHEN 'police' THEN 3 ELSE 4 END
    LIMIT 1
) v ON true
WHERE i.status IN ('active', 'verifying');

-- Emergency vehicles enroute
CREATE OR REPLACE VIEW emergency_vehicles_enroute AS
SELECT
    v.id,
    v.callsign,
    v.type,
    v.current_location,
    v.heading,
    v.speed_kmh,
    v.destination,
    v.current_route,
    v.priority_level,
    v.status,
    v.capabilities,
    rr.eta_seconds,
    rr.distance_meters,
    rr.origin,
    u.name as driver_name,
    u.phone as driver_phone
FROM vehicles v
LEFT JOIN route_requests rr ON rr.vehicle_id = v.id AND rr.status IN ('accepted', 'active')
LEFT JOIN users u ON u.id = v.driver_id
WHERE v.type IN ('ambulance', 'fire', 'police', 'vip')
  AND v.status IN ('enroute', 'on_scene');

-- Intersection status with preemption
CREATE OR REPLACE VIEW intersection_status AS
SELECT
    i.id,
    i.name,
    i.geometry,
    i.phase_plan,
    i.current_phase,
    i.phase_start_time,
    i.detectors,
    i.preemption_active,
    i.preemption_type,
    i.preemption_vehicle_id,
    v.callsign as preemption_vehicle,
    v.type as preemption_vehicle_type,
    i.controller_type,
    i.is_online,
    i.last_update,
    EXTRACT(EPOCH FROM (NOW() - i.last_update)) as seconds_since_update
FROM intersections i
LEFT JOIN vehicles v ON v.id = i.preemption_vehicle_id;

-- Route performance analytics
CREATE OR REPLACE VIEW route_performance AS
SELECT
    DATE_TRUNC('hour', created_at) as hour_bucket,
    priority_level,
    COUNT(*) as total_requests,
    AVG(eta_seconds) as avg_eta,
    AVG(distance_meters) as avg_distance,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_count,
    COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled_count,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY eta_seconds) as median_eta,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY eta_seconds) as p95_eta
FROM route_requests
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY hour_bucket, priority_level
ORDER BY hour_bucket DESC, priority_level;

-- Incident response times
CREATE OR REPLACE VIEW incident_response_times AS
SELECT
    i.id,
    i.type,
    i.severity,
    i.created_at as incident_created,
    MIN(v.last_heartbeat) FILTER (WHERE v.id = ANY(i.assigned_services)) as first_dispatch,
    MAX(v.last_heartbeat) FILTER (WHERE v.id = ANY(i.assigned_services)) as last_arrival,
    i.cleared_at,
    EXTRACT(EPOCH FROM (MIN(v.last_heartbeat) FILTER (WHERE v.id = ANY(i.assigned_services)) - i.created_at)) / 60 as first_response_minutes,
    EXTRACT(EPOCH FROM (i.cleared_at - i.created_at)) / 60 as total_duration_minutes
FROM incidents i
LEFT JOIN vehicles v ON v.id = ANY(i.assigned_services)
WHERE i.status IN ('cleared', 'archived')
  AND i.created_at > NOW() - INTERVAL '30 days'
GROUP BY i.id, i.type, i.severity, i.created_at, i.cleared_at;

-- Public route alternatives (for displaying 2nd best)
CREATE OR REPLACE VIEW public_route_alternatives AS
SELECT
    rr.id,
    rr.user_id,
    rr.origin,
    rr.destination,
    rr.assigned_route as best_route,
    rr.alternative_routes,
    rr.status,
    rr.created_at,
    CASE
        WHEN array_length(rr.alternative_routes, 1) > 0 THEN rr.alternative_routes[1]
        ELSE NULL
    END as second_best_route,
    CASE
        WHEN array_length(rr.alternative_routes, 1) > 1 THEN rr.alternative_routes[2]
        ELSE NULL
    END as third_best_route
FROM route_requests rr
WHERE rr.priority_level = 0
  AND rr.status IN ('calculated', 'accepted', 'active')
  AND rr.created_at > NOW() - INTERVAL '24 hours';

-- Materialized view for congestion heatmap (refresh every 5 min) - simplified
CREATE MATERIALIZED VIEW congestion_heatmap AS
SELECT 
    ST_SnapToGrid(geometry, 0.002) as grid_cell,
    AVG(congestion_level) as avg_congestion,
    COUNT(*) as segment_count
FROM live_congestion
WHERE last_updated > NOW() - INTERVAL '10 minutes'
GROUP BY grid_cell;

CREATE UNIQUE INDEX idx_congestion_heatmap_hex ON congestion_heatmap (grid_cell);

-- Function to refresh congestion heatmap
CREATE OR REPLACE FUNCTION refresh_congestion_heatmap()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY congestion_heatmap;
END;
$$ LANGUAGE plpgsql;

-- Schedule: SELECT cron.schedule('refresh-heatmap', '*/5 * * * *', 'SELECT refresh_congestion_heatmap();');
-- Requires pg_cron extension