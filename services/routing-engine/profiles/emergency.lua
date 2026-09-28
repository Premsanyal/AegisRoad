-- Emergency vehicle routing profile for OSRM
-- Based on car.lua with modifications for emergency vehicles

api_version = 4

Set = require('lib/set')
Sequence = require('lib/sequence')
Handlers = require('lib/way_handlers')
find_access_tag = require('lib/access').find_access_tag
limit = require('lib/maxspeed').limit

-- Emergency vehicles can use all roads including restricted ones
function setup()
  return {
    properties = {
      max_speed_for_map_matching = 180/3.6, -- 180 km/h for emergency
      weight_name = 'emergency',
      -- Emergency vehicles can use bus lanes, emergency lanes
      use_turn_restrictions = false,
      continue_straight_at_waypoint = true,
      turn_penalty = 5,
      turn_bias = 1.0,
    },

    default_mode = mode.driving,
    modes = {
      driving = Mode('driving', 1),
    },

    default_speed = 50,
    oneway_handling = true,
    side_road_multiplier = 0.8,
    turn_penalty = 5,
    speed_reduction = 0.8,
  }
end

function process_node(profile, node, result)
  -- Emergency vehicles ignore most access restrictions
  -- Only respect physical barriers
  local access = find_access_tag(node, profile, { 'emergency', 'motorcar', 'vehicle', 'access' })
  if access and access ~= '' then
    if access == 'no' or access == 'private' then
      result.barrier = true
    end
  end
end

function process_way(profile, way, result)
  -- Emergency vehicles can use almost any road
  local highway = way:get_value_by_key('highway')
  local access = find_access_tag(way, profile, { 'emergency', 'motorcar', 'vehicle', 'access' })

  -- Only reject if physically impossible
  if access == 'no' then
    return
  end

  -- Speed limits - emergency can exceed but we cap for safety
  local maxspeed = limit(way, profile, 160) -- Cap at 160 km/h
  result.forward_speed = maxspeed
  result.backward_speed = maxspeed

  -- Road classification for emergency routing
  if highway == 'motorway' or highway == 'motorway_link' then
    result.forward_speed = math.min(result.forward_speed, 160)
    result.backward_speed = math.min(result.backward_speed, 160)
    result.forward_rate = 1.0
    result.backward_rate = 1.0
  elseif highway == 'trunk' or highway == 'trunk_link' then
    result.forward_speed = math.min(result.forward_speed, 120)
    result.backward_speed = math.min(result.backward_speed, 120)
    result.forward_rate = 1.0
    result.backward_rate = 1.0
  elseif highway == 'primary' or highway == 'primary_link' then
    result.forward_speed = math.min(result.forward_speed, 100)
    result.backward_speed = math.min(result.backward_speed, 100)
    result.forward_rate = 1.0
    result.backward_rate = 1.0
  elseif highway == 'secondary' or highway == 'secondary_link' then
    result.forward_speed = math.min(result.forward_speed, 80)
    result.backward_speed = math.min(result.backward_speed, 80)
    result.forward_rate = 0.95
    result.backward_rate = 0.95
  elseif highway == 'tertiary' or highway == 'tertiary_link' then
    result.forward_speed = math.min(result.forward_speed, 60)
    result.backward_speed = math.min(result.backward_speed, 60)
    result.forward_rate = 0.9
    result.backward_rate = 0.9
  elseif highway == 'residential' or highway == 'living_street' then
    result.forward_speed = math.min(result.forward_speed, 40)
    result.backward_speed = math.min(result.backward_speed, 40)
    result.forward_rate = 0.8
    result.backward_rate = 0.8
  elseif highway == 'service' then
    result.forward_speed = math.min(result.forward_speed, 30)
    result.backward_speed = math.min(result.backward_speed, 30)
    result.forward_rate = 0.7
    result.backward_rate = 0.7
  elseif highway == 'track' then
    result.forward_speed = math.min(result.forward_speed, 20)
    result.backward_speed = math.min(result.backward_speed, 20)
    result.forward_rate = 0.5
    result.backward_rate = 0.5
  else
    -- Unknown highway type - allow but with penalty
    result.forward_speed = math.min(result.forward_speed, 30)
    result.backward_speed = math.min(result.backward_speed, 30)
    result.forward_rate = 0.6
    result.backward_rate = 0.6
  end

  -- Emergency vehicles can use bus lanes and emergency lanes
  local lanes = way:get_value_by_key('lanes')
  local bus_lane = way:get_value_by_key('bus_lane')
  local emergency_lane = way:get_value_by_key('emergency_lane')

  if bus_lane == 'yes' or emergency_lane == 'yes' then
    result.forward_rate = result.forward_rate * 1.1 -- Slight bonus for dedicated lanes
    result.backward_rate = result.backward_rate * 1.1
  end

  -- Handle oneway
  local oneway = way:get_value_by_key('oneway')
  if oneway == 'yes' or oneway == '1' or oneway == 'true' then
    result.backward_speed = 0
  elseif oneway == '-1' then
    result.forward_speed = 0
  end

  -- Weight for emergency routing - prioritize speed
  result.weight = 1.0 / result.forward_rate
end

function process_turn(profile, turn)
  -- Minimal turn penalty for emergency vehicles
  if turn.has_traffic_light then
    turn.duration = turn.duration + 2 -- Only 2 seconds penalty
  elseif turn.is_u_turn then
    turn.duration = turn.duration + 5 -- U-turns take longer
  else
    turn.duration = turn.duration + 1
  end

  -- Emergency vehicles can ignore some turn restrictions
  if turn.angle > 150 then
    turn.weight = turn.weight * 0.5 -- Sharp turns penalized less
  end
end

return {
  setup = setup,
  process_way = process_way,
  process_node = process_node,
  process_turn = process_turn,
}