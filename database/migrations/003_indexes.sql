-- 003_indexes.sql
-- Additional performance indexes and constraints

-- Partial indexes for common queries
CREATE INDEX idx_road_segments_active_congestion ON road_segments (congestion_level) WHERE congestion_level > 2;
CREATE INDEX idx_incidents_active ON incidents (created_at DESC) WHERE status IN ('active', 'verifying');
CREATE INDEX idx_vehicles_active_emergency ON vehicles (current_location) WHERE type IN ('ambulance', 'fire', 'police') AND status = 'enroute';
CREATE INDEX idx_route_requests_active ON route_requests (created_at DESC) WHERE status IN ('calculated', 'accepted', 'active');

-- Composite indexes for routing queries
CREATE INDEX idx_road_segments_routing ON road_segments (highway_type, priority_tier, congestion_level);

-- Full-text search for road names
ALTER TABLE road_segments ADD COLUMN IF NOT EXISTS name_tsv tsvector GENERATED ALWAYS AS (to_tsvector('english', COALESCE(name, ''))) STORED;
CREATE INDEX idx_road_segments_name_fts ON road_segments USING GIN (name_tsv);

-- Partition traffic_speeds by day (already done in 002, but add compression policy)
-- Note: Requires TimescaleDB 2.0+
-- ALTER TABLE traffic_speeds SET (timescaledb.compress, timescaledb.compress_segmentby = 'segment_id');
-- SELECT add_compression_policy('traffic_speeds', INTERVAL '7 days');

-- Constraints
ALTER TABLE road_segments ADD CONSTRAINT chk_congestion_level CHECK (congestion_level BETWEEN 0 AND 5);
ALTER TABLE incidents ADD CONSTRAINT chk_severity CHECK (severity BETWEEN 1 AND 5);
ALTER TABLE vehicles ADD CONSTRAINT chk_priority CHECK (priority_level BETWEEN 0 AND 3);
ALTER TABLE users ADD CONSTRAINT chk_role CHECK (role IN ('public', 'emergency', 'operator', 'admin'));
ALTER TABLE route_requests ADD CONSTRAINT chk_route_priority CHECK (priority_level BETWEEN 0 AND 3);
ALTER TABLE intersections ADD CONSTRAINT chk_phase CHECK (current_phase >= 0);

-- Foreign key for assigned_services in incidents (array of vehicle IDs)
-- Handled at application level for performance

-- Statistics targets for query planner
ALTER TABLE road_segments ALTER COLUMN highway_type SET STATISTICS 1000;
ALTER TABLE road_segments ALTER COLUMN congestion_level SET STATISTICS 1000;
ALTER TABLE incidents ALTER COLUMN status SET STATISTICS 1000;
ALTER TABLE incidents ALTER COLUMN type SET STATISTICS 1000;
ALTER TABLE vehicles ALTER COLUMN type SET STATISTICS 1000;
ALTER TABLE vehicles ALTER COLUMN status SET STATISTICS 1000;