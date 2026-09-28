-- 005_seed.sql
-- Initial seed data for Bengaluru prototype

-- Insert major road types for Bengaluru
INSERT INTO road_segments (osm_id, geometry, highway_type, name, max_speed, lanes, priority_tier) VALUES
-- Outer Ring Road (ORR)
(1001, ST_GeomFromText('LINESTRING(77.55 12.92, 77.60 12.95, 77.65 12.97)', 4326), 'primary', 'Outer Ring Road', 80, 4, 1),
(1002, ST_GeomFromText('LINESTRING(77.65 12.97, 77.70 12.95, 77.75 12.92)', 4326), 'primary', 'Outer Ring Road', 80, 4, 1),

-- Major arterials
(2001, ST_GeomFromText('LINESTRING(77.55 12.97, 77.60 12.97)', 4326), 'primary', 'Old Airport Road', 60, 4, 1),
(2002, ST_GeomFromText('LINESTRING(77.60 12.97, 77.65 12.97)', 4326), 'primary', 'Sarjapur Road', 60, 4, 1),
(2003, ST_GeomFromText('LINESTRING(77.57 12.95, 77.62 12.95)', 4326), 'secondary', 'Koramangala Inner Ring', 50, 4, 0),
(2004, ST_GeomFromText('LINESTRING(77.59 12.93, 77.64 12.93)', 4326), 'secondary', 'HSR Layout Main', 50, 4, 0),
(2005, ST_GeomFromText('LINESTRING(77.56 13.00, 77.61 13.00)', 4326), 'primary', 'Hebbal Flyover', 80, 6, 1),

-- Metro corridor roads (priority for public transit)
(3001, ST_GeomFromText('LINESTRING(77.55 12.97, 77.58 12.97)', 4326), 'secondary', 'MG Road', 40, 4, 1),
(3002, ST_GeomFromText('LINESTRING(77.58 12.97, 77.61 12.97)', 4326), 'secondary', 'Brigade Road', 40, 4, 1),

-- Residential / local
(4001, ST_GeomFromText('LINESTRING(77.62 12.93, 77.63 12.93)', 4326), 'residential', '1st Main HSR', 30, 2, 0),
(4002, ST_GeomFromText('LINESTRING(77.61 12.94, 77.62 12.94)', 4326), 'residential', '2nd Cross Koramangala', 30, 2, 0),

-- Emergency corridors (reserved for emergency vehicles)
(5001, ST_GeomFromText('LINESTRING(77.58 12.95, 77.59 12.95)', 4326), 'tertiary', 'Hospital Access Road', 40, 2, 2),
(5002, ST_GeomFromText('LINESTRING(77.60 12.96, 77.61 12.96)', 4326), 'tertiary', 'Fire Station Access', 40, 2, 2)
ON CONFLICT (osm_id) DO NOTHING;

-- Insert sample intersections (major junctions)
INSERT INTO intersections (id, name, geometry, phase_plan, controller_type) VALUES
('BLR-001', 'Silk Board Junction', ST_GeomFromText('POINT(77.62 12.91)', 4326),
 '{"phases": [{"id": 1, "directions": ["N-S"], "duration": 60}, {"id": 2, "directions": ["E-W"], "duration": 60}, {"id": 3, "directions": ["N-S-left"], "duration": 20}, {"id": 4, "directions": ["E-W-left"], "duration": 20}], "cycle": 160, "offset": 0}', 'SCATS'),

('BLR-002', 'Marathahalli Bridge', ST_GeomFromText('POINT(77.70 12.96)', 4326),
 '{"phases": [{"id": 1, "directions": ["N-S"], "duration": 70}, {"id": 2, "directions": ["E-W"], "duration": 50}, {"id": 3, "directions": ["pedestrian"], "duration": 30}], "cycle": 150, "offset": 30}', 'SCATS'),

('BLR-003', 'Hebbal Flyover Junction', ST_GeomFromText('POINT(77.59 13.03)', 4326),
 '{"phases": [{"id": 1, "directions": ["NH44"], "duration": 90}, {"id": 2, "directions": ["service-road"], "duration": 40}, {"id": 3, "directions": ["ramp"], "duration": 30}], "cycle": 160, "offset": 60}', 'SCATS'),

('BLR-004', 'Koramangala 80ft Road', ST_GeomFromText('POINT(77.62 12.93)', 4326),
 '{"phases": [{"id": 1, "directions": ["E-W"], "duration": 50}, {"id": 2, "directions": ["N-S"], "duration": 50}, {"id": 3, "directions": ["pedestrian"], "duration": 25}], "cycle": 125, "offset": 10}', 'SCATS'),

('BLR-005', 'Electronic City Phase 1', ST_GeomFromText('POINT(77.67 12.84)', 4326),
 '{"phases": [{"id": 1, "directions": ["N-S"], "duration": 60}, {"id": 2, "directions": ["E-W"], "duration": 60}], "cycle": 120, "offset": 0}', 'SCATS')
ON CONFLICT (id) DO NOTHING;

-- Insert sample cameras
INSERT INTO cameras (id, name, geometry, stream_url, stream_type, direction, intersection_id) VALUES
('CAM-001', 'Silk Board North', ST_GeomFromText('POINT(77.621 12.912)', 4326), 'rtsp://cameras.bbtp.gov.in/cam001', 'rtsp', 0, 'BLR-001'),
('CAM-002', 'Silk Board South', ST_GeomFromText('POINT(77.619 12.908)', 4326), 'rtsp://cameras.bbtp.gov.in/cam002', 'rtsp', 180, 'BLR-001'),
('CAM-003', 'Marathahalli East', ST_GeomFromText('POINT(77.702 12.958)', 4326), 'rtsp://cameras.bbtp.gov.in/cam003', 'rtsp', 90, 'BLR-002'),
('CAM-004', 'Hebbal Flyover', ST_GeomFromText('POINT(77.592 13.032)', 4326), 'rtsp://cameras.bbtp.gov.in/cam004', 'rtsp', 45, 'BLR-003'),
('CAM-005', 'Koramangala 80ft', ST_GeomFromText('POINT(77.622 12.932)', 4326), 'rtsp://cameras.bbtp.gov.in/cam005', 'rtsp', 270, 'BLR-004')
ON CONFLICT (id) DO NOTHING;

-- Insert sample emergency vehicles (for testing)
INSERT INTO vehicles (id, type, registration, callsign, priority_level, status, capabilities, current_location) VALUES
('aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee1', 'ambulance', 'KA01AMB001', 'AMB-001', 2, 'idle', '{"als", "bls"}', ST_GeomFromText('POINT(77.60 12.97)', 4326)::geography),
('aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee2', 'ambulance', 'KA01AMB002', 'AMB-002', 2, 'idle', '{"bls"}', ST_GeomFromText('POINT(77.65 12.93)', 4326)::geography),
('aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee3', 'fire', 'KA01FIRE01', 'ENG-01', 2, 'idle', '{"pump", "ladder", "hazmat"}', ST_GeomFromText('POINT(77.58 12.95)', 4326)::geography),
('aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee4', 'fire', 'KA01FIRE02', 'ENG-02', 2, 'idle', '{"pump", "tanker"}', ST_GeomFromText('POINT(77.70 12.96)', 4326)::geography),
('aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee5', 'police', 'KA01POL001', 'PCR-01', 2, 'idle', '{"patrol", "interceptor"}', ST_GeomFromText('POINT(77.62 12.93)', 4326)::geography),
('aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee6', 'police', 'KA01POL002', 'PCR-02', 2, 'idle', '{"patrol"}', ST_GeomFromText('POINT(77.59 13.00)', 4326)::geography)
ON CONFLICT (id) DO NOTHING;

-- Insert sample users (passwords handled by auth service)
INSERT INTO users (id, role, phone, email, name, vehicle_id) VALUES
('bbbbbbbb-cccc-dddd-eeee-fffffffffff1', 'emergency', '+919876543210', 'ambulance1@aegisroad.in', 'Dr. Rajesh Kumar', 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee1'),
('bbbbbbbb-cccc-dddd-eeee-fffffffffff2', 'emergency', '+919876543211', 'ambulance2@aegisroad.in', 'Dr. Priya Sharma', 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee2'),
('bbbbbbbb-cccc-dddd-eeee-fffffffffff3', 'emergency', '+919876543212', 'fire1@aegisroad.in', 'Station Officer Ramesh', 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee3'),
('bbbbbbbb-cccc-dddd-eeee-fffffffffff4', 'emergency', '+919876543213', 'fire2@aegisroad.in', 'Station Officer Lakshmi', 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee4'),
('bbbbbbbb-cccc-dddd-eeee-fffffffffff5', 'emergency', '+919876543214', 'police1@aegisroad.in', 'Inspector Kumar', 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeee5'),
('bbbbbbbb-cccc-dddd-eeee-fffffffffff6', 'operator', '+919876543215', 'operator1@aegisroad.in', 'Traffic Operator 1', NULL),
('bbbbbbbb-cccc-dddd-eeee-fffffffffff7', 'operator', '+919876543216', 'operator2@aegisroad.in', 'Traffic Operator 2', NULL),
('bbbbbbbb-cccc-dddd-eeee-fffffffffff8', 'admin', '+919876543217', 'admin@aegisroad.in', 'System Admin', NULL)
ON CONFLICT (id) DO NOTHING;

-- Sample traffic speeds (simulated current conditions)
INSERT INTO traffic_speeds (segment_id, speed_kmh, confidence, source, timestamp) VALUES
(1001, 45, 0.9, 'tomtom', NOW() - INTERVAL '2 minutes'),
(1002, 35, 0.85, 'here', NOW() - INTERVAL '3 minutes'),
(2001, 25, 0.95, 'sensor', NOW() - INTERVAL '1 minute'),
(2002, 15, 0.9, 'crowd', NOW() - INTERVAL '5 minutes'),
(2003, 40, 0.8, 'tomtom', NOW() - INTERVAL '2 minutes'),
(2004, 30, 0.85, 'here', NOW() - INTERVAL '3 minutes'),
(2005, 65, 0.9, 'sensor', NOW() - INTERVAL '1 minute'),
(3001, 20, 0.9, 'tomtom', NOW() - INTERVAL '2 minutes'),
(3002, 18, 0.85, 'here', NOW() - INTERVAL '3 minutes'),
(4001, 25, 0.7, 'crowd', NOW() - INTERVAL '5 minutes'),
(4002, 28, 0.75, 'crowd', NOW() - INTERVAL '4 minutes'),
(5001, 35, 0.95, 'sensor', NOW() - INTERVAL '1 minute'),
(5002, 38, 0.95, 'sensor', NOW() - INTERVAL '1 minute')
ON CONFLICT DO NOTHING;

-- Update road segment congestion levels based on latest speeds
UPDATE road_segments r SET congestion_level = (
    SELECT CASE
        WHEN COALESCE(t.speed_kmh, r.max_speed) / NULLIF(r.max_speed, 0) > 0.8 THEN 0
        WHEN COALESCE(t.speed_kmh, r.max_speed) / NULLIF(r.max_speed, 0) > 0.6 THEN 1
        WHEN COALESCE(t.speed_kmh, r.max_speed) / NULLIF(r.max_speed, 0) > 0.4 THEN 2
        WHEN COALESCE(t.speed_kmh, r.max_speed) / NULLIF(r.max_speed, 0) > 0.2 THEN 3
        WHEN COALESCE(t.speed_kmh, r.max_speed) / NULLIF(r.max_speed, 0) > 0.1 THEN 4
        ELSE 5
    END
    FROM traffic_speeds t
    WHERE t.segment_id = r.osm_id
    ORDER BY t.timestamp DESC
    LIMIT 1
);