package models

import "time"

type PhasePlan struct {
	Phases []Phase `json:"phases"`
	Cycle  int     `json:"cycle"`  // seconds
	Offset int     `json:"offset"` // seconds from cycle start
}

type Phase struct {
	ID          int      `json:"id"`
	Directions  []string `json:"directions"`  // e.g., ["N-S", "E-W-left"]
	Duration    int      `json:"duration"`    // seconds
	MinDuration int      `json:"min_duration,omitempty"`
	MaxDuration int      `json:"max_duration,omitempty"`
}

type Intersection struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Latitude        float64    `json:"latitude"`
	Longitude       float64    `json:"longitude"`
	PhasePlan       PhasePlan  `json:"phase_plan"`
	CurrentPhase    int        `json:"current_phase"`
	PhaseStartTime  time.Time  `json:"phase_start_time"`
	PhaseEndTime    time.Time  `json:"phase_end_time"`
	Detectors       []Detector `json:"detectors"`
	PreemptionActive bool      `json:"preemption_active"`
	PreemptionType   string    `json:"preemption_type,omitempty"` // emergency, transit
	PreemptionVehicleID string  `json:"preemption_vehicle_id,omitempty"`
	ControllerType   string    `json:"controller_type"`
	IsOnline         bool       `json:"is_online"`
	LastUpdate       time.Time  `json:"last_update"`
}

type Detector struct {
	ID            string  `json:"id"`
	IntersectionID string `json:"intersection_id"`
	Approach      string  `json:"approach"` // N, S, E, W
	Lane          int     `json:"lane"`
	Occupancy     float64 `json:"occupancy"` // 0-1
	Volume        int     `json:"volume"`    // vehicles per hour
	Speed         float64 `json:"speed"`     // km/h
	Timestamp     time.Time `json:"timestamp"`
}

type PreemptionRequest struct {
	ID              string    `json:"id"`
	VehicleID       string    `json:"vehicle_id"`
	VehicleType     string    `json:"vehicle_type"` // ambulance, fire, police
	IntersectionID  string    `json:"intersection_id"`
	Approach        string    `json:"approach"`     // direction vehicle is coming from
	RequestedPhase  int       `json:"requested_phase"`
	Priority        int       `json:"priority"`     // 1-10
	Status          string    `json:"status"`       // requested, granted, active, cancelled
	RequestedAt     time.Time `json:"requested_at"`
	GrantedAt       *time.Time `json:"granted_at,omitempty"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
}

type IntersectionStatus struct {
	IntersectionID  string    `json:"intersection_id"`
	CurrentPhase    int       `json:"current_phase"`
	PhaseElapsed    int       `json:"phase_elapsed"`    // seconds in current phase
	PhaseRemaining  int       `json:"phase_remaining"`  // seconds until next phase
	NextPhase       int       `json:"next_phase"`
	PreemptionActive bool      `json:"preemption_active"`
	PreemptionType   string    `json:"preemption_type,omitempty"`
	Detectors       []Detector `json:"detectors"`
	Timestamp       time.Time `json:"timestamp"`
}

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}