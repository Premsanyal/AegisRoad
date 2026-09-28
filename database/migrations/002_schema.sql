-- 002_schema.sql
-- Core schema for AegisRoad traffic management system

-- Road network segments (from OSM)
CREATE TABLE road_segments (
    id BIGSERIAL PRIMARY KEY,
    osm_id BIGINT UNIQUE NOT NULL,
    geometry LINESTRING(4326) NOT NULL,
    highway_type TEXT NOT NULL,           -- motorway, primary, secondary, tertiary, residential, etc.
    name TEXT,
    max_speed INT DEFAULT 50,             -- km/h
    lanes INT DEFAULT 2,
    is_oneway BOOLEAN DEFAULT FALSE,
    priority_tier INT DEFAULT 0,          -- 0=public, 1=reserved_peak, 2=emergency_only
    surface TEXT,
    bridge BOOLEAN DEFAULT FALSE,
    tunnel BOOLEAN DEFAULT FALSE,
    congestion_level INT DEFAULT 0,       -- 0-5 real-time (0=free, 5=gridlock)
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Live traffic speeds (from APIs + crowdsourcing)
CREATE TABLE traffic_speeds (
    segment_id BIGINT NOT NULL REFERENCES road_segments(id) ON DELETE CASCADE,
    speed_kmh REAL NOT NULL,
    confidence REAL DEFAULT 1.0,          -- 0-1 data quality
    source TEXT NOT NULL,                 -- 'tomtom', 'here', 'crowd', 'sensor', 'ml'
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (segment_id, timestamp)
);

-- Convert to hypertable for time-series optimization
SELECT create_hypertable('traffic_speeds', 'timestamp', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);

-- Incidents (accidents, breakdowns, construction, events)
CREATE TABLE incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL,                   -- accident, breakdown, construction, flood, event, hazard
    severity INT NOT NULL DEFAULT 1,      -- 1-5 (5=critical)
    geometry POINT(4326) NOT NULL,
    affected_segments BIGINT[] DEFAULT '{}',
    description TEXT,
    reported_by TEXT NOT NULL,            -- 'operator', 'public', 'sensor', 'ml', 'camera'
    reporter_id UUID,                     -- user/device ID
    status TEXT NOT NULL DEFAULT 'active', -- active, verifying, cleared, archived
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cleared_at TIMESTAMPTZ,
    assigned_services UUID[] DEFAULT '{}', -- dispatched unit IDs
    metadata JSONB DEFAULT '{}'
);

-- Vehicles (public + emergency + operators)
CREATE TABLE vehicles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL,                   -- 'public', 'ambulance', 'fire', 'police', 'vip', 'operator'
    registration TEXT UNIQUE,
    callsign TEXT,                        -- for emergency: "AMB-001", "ENG-05"
    current_location GEOGRAPHY(POINT, 4326),
    heading REAL,                         -- degrees 0-360
    speed_kmh REAL DEFAULT 0,
    destination GEOGRAPHY(POINT, 4326),
    current_route JSONB,                  -- GeoJSON LineString with metadata
    alternative_routes JSONB[],           -- array of backup routes
    priority_level INT DEFAULT 0,         -- 0=public, 1=high, 2=emergency, 3=vip
    status TEXT NOT NULL DEFAULT 'idle',  -- idle, enroute, on_scene, returning, maintenance
    driver_id UUID,
    capabilities TEXT[],                  -- ['als', 'bls', 'hazmat', 'command']
    battery_level INT,                    -- for EVs
    last_heartbeat TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Users (3 roles + admin)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role TEXT NOT NULL CHECK (role IN ('public', 'emergency', 'operator', 'admin')),
    phone TEXT UNIQUE,
    email TEXT UNIQUE,
    name TEXT,
    vehicle_id UUID REFERENCES vehicles(id), -- for emergency/operator personnel
    preferences JSONB DEFAULT '{}',         -- notification settings, map prefs, etc.
    fcm_token TEXT,                         -- for push notifications
    is_active BOOLEAN DEFAULT TRUE,
    last_login TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Route requests & history
CREATE TABLE route_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    vehicle_id UUID REFERENCES vehicles(id),
    origin GEOGRAPHY(POINT, 4326) NOT NULL,
    destination GEOGRAPHY(POINT, 4326) NOT NULL,
    priority_level INT NOT NULL DEFAULT 0,
    assigned_route JSONB,                   -- GeoJSON LineString
    alternative_routes JSONB[] DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'calculated', -- calculated, accepted, active, completed, cancelled
    eta_seconds INT,
    distance_meters INT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    accepted_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

-- Signal intersections (SCATS-emulated)
CREATE TABLE intersections (
    id TEXT PRIMARY KEY,                    -- e.g., "BLR-001"
    name TEXT,
    geometry POINT(4326) NOT NULL,
    phase_plan JSONB NOT NULL,              -- current timing plan
    current_phase INT DEFAULT 0,
    phase_start_time TIMESTAMPTZ DEFAULT NOW(),
    detectors JSONB DEFAULT '[]',           -- loop detector data
    preemption_active BOOLEAN DEFAULT FALSE,
    preemption_type TEXT,                   -- 'emergency', 'transit', 'none'
    preemption_vehicle_id UUID REFERENCES vehicles(id),
    controller_type TEXT DEFAULT 'SCATS',   -- SCATS, SCOOT, fixed
    is_online BOOLEAN DEFAULT TRUE,
    last_update TIMESTAMPTZ DEFAULT NOW()
);

-- Camera feeds (BBMP traffic cameras)
CREATE TABLE cameras (
    id TEXT PRIMARY KEY,
    name TEXT,
    geometry POINT(4326) NOT NULL,
    stream_url TEXT,                        -- RTSP/HLS URL
    stream_type TEXT,                       -- 'rtsp', 'hls', 'mjpeg'
    direction INT,                          -- degrees
    intersection_id TEXT REFERENCES intersections(id),
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}'
);

-- Alerts/broadcasts from operators
CREATE TABLE alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL,                     -- 'incident', 'congestion', 'road_closure', 'weather', 'vip', 'test'
    severity TEXT NOT NULL,                 -- 'info', 'warning', 'critical', 'emergency'
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    geometry GEOGRAPHY(POLYGON, 4326),     -- affected area
    target_roles TEXT[] DEFAULT '{public}', -- who receives: public, emergency, operator
    created_by UUID REFERENCES users(id),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE
);

-- Audit log for all critical actions
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,                   -- 'route_request', 'incident_create', 'signal_override', 'dispatch'
    resource_type TEXT,                     -- 'route', 'incident', 'vehicle', 'intersection'
    resource_id TEXT,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_road_segments_geom ON road_segments USING GIST (geometry);
CREATE INDEX idx_road_segments_highway ON road_segments (highway_type);
CREATE INDEX idx_road_segments_priority ON road_segments (priority_tier);
CREATE INDEX idx_road_segments_congestion ON road_segments (congestion_level);

CREATE INDEX idx_traffic_speeds_segment_time ON traffic_speeds (segment_id, timestamp DESC);
CREATE INDEX idx_traffic_speeds_source ON traffic_speeds (source);

CREATE INDEX idx_incidents_geom ON incidents USING GIST (geometry);
CREATE INDEX idx_incidents_status ON incidents (status);
CREATE INDEX idx_incidents_type_severity ON incidents (type, severity);
CREATE INDEX idx_incidents_created ON incidents (created_at DESC);

CREATE INDEX idx_vehicles_location ON vehicles USING GIST (current_location);
CREATE INDEX idx_vehicles_type_status ON vehicles (type, status);
CREATE INDEX idx_vehicles_driver ON vehicles (driver_id);

CREATE INDEX idx_users_role ON users (role);
CREATE INDEX idx_users_phone ON users (phone);
CREATE INDEX idx_users_vehicle ON users (vehicle_id);

CREATE INDEX idx_route_requests_user ON route_requests (user_id, created_at DESC);
CREATE INDEX idx_route_requests_vehicle ON route_requests (vehicle_id, created_at DESC);
CREATE INDEX idx_route_requests_status ON route_requests (status);

CREATE INDEX idx_intersections_geom ON intersections USING GIST (geometry);
CREATE INDEX idx_intersections_online ON intersections (is_online);

CREATE INDEX idx_cameras_geom ON cameras USING GIST (geometry);
CREATE INDEX idx_cameras_intersection ON cameras (intersection_id);

CREATE INDEX idx_alerts_active ON alerts (is_active, expires_at) WHERE is_active = TRUE;
CREATE INDEX idx_alerts_geom ON alerts USING GIST (geometry);

CREATE INDEX idx_audit_log_user ON audit_log (user_id, created_at DESC);
CREATE INDEX idx_audit_log_resource ON audit_log (resource_type, resource_id);
CREATE INDEX idx_audit_log_created ON audit_log (created_at DESC);

-- Updated_at triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_road_segments_updated_at BEFORE UPDATE ON road_segments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_incidents_updated_at BEFORE UPDATE ON incidents FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_vehicles_updated_at BEFORE UPDATE ON vehicles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();