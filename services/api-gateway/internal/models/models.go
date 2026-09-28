package models

import (
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
)

type Point struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

type RoutingProfile int

const (
	ProfileCar RoutingProfile = iota
	ProfileBike
	ProfileFoot
	ProfileEmergency
)

type RouteRequest struct {
	RequestID          string
	PriorityLevel      int32
	Origin             Point
	Destination        Point
	VehicleID          string
	VehicleType        string
	AvoidSegments      []int64
	ReturnAlternatives bool
	MaxAlternatives    int
	Profile            RoutingProfile
}

type RouteResponse struct {
	RequestID          string
	PrimaryRoute       *Route
	AlternativeRoutes  []*Route
	Status             RouteStatus
	ErrorMessage       string
	CalculatedAt       int64
}

type RouteStatus int

const (
	RouteStatusOK RouteStatus = iota
	RouteStatusNoRoute
	RouteStatusPartial
	RouteStatusError
)

type Route struct {
	RouteID                     string
	Geometry                    []Point
	Segments                    []SegmentInfo
	DistanceMeters              int64
	DurationSeconds             int32
	DurationWithTrafficSeconds  int32
	CongestionLevel             int32
	UsesReservedLanes           bool
	Warnings                    []IncidentWarning
	Metadata                    map[string]interface{}
}

type SegmentInfo struct {
	SegmentID       int64
	OSMID           int64
	Name            string
	HighwayType     string
	MaxSpeed        int32
	CurrentSpeed    int32
	CongestionLevel int32
	LengthMeters    int64
	DurationSeconds int32
	IsReserved      bool
	IsOneway        bool
}

type IncidentWarning struct {
	IncidentID   string
	Location     Point
	IncidentType string
	Severity     int32
	Description  string
	DelaySeconds int32
}

type RouteUpdate struct {
	RequestID       string
	UpdateType      RouteUpdateType
	UpdatedRoute    *Route
	NewIncident     *IncidentWarning
	CongestionChange *CongestionChange
	Timestamp       int64
}

type RouteUpdateType int

const (
	RouteUpdateRecalculated RouteUpdateType = iota
	RouteUpdateIncident
	RouteUpdateCongestion
	RouteUpdateArrived
)

type CongestionChange struct {
	SegmentID  int64
	OldLevel   int32
	NewLevel   int32
}

type CorridorRequest struct {
	ReservationID  string
	VehicleID      string
	CorridorPath   []Point
	LookaheadMeters int32
	ExpiresAt      int64
	PriorityLevel  int32
}

type CorridorResponse struct {
	ReservationID      string
	Success            bool
	ErrorMessage       string
	ReservedSegmentIDs []int64
	ExpiresAt          int64
}

// Domain models for handlers

type User struct {
	ID        uuid.UUID
	Role      string
	Phone     string
	Email     string
	Name      string
	VehicleID *uuid.UUID
	Preferences map[string]interface{}
	FCMToken  string
	IsActive  bool
	CreatedAt pgtype.Timestamptz
	UpdatedAt pgtype.Timestamptz
}

type Vehicle struct {
	ID              uuid.UUID
	Type            string
	Registration    string
	CallSign        string
	CurrentLocation Point
	Heading         float64
	SpeedKmh        float64
	Destination     Point
	CurrentRoute    map[string]interface{}
	PriorityLevel   int32
	Status          string
	DriverID        *uuid.UUID
	Capabilities    []string
	BatteryLevel    int32
	LastHeartbeat   pgtype.Timestamptz
}

type Incident struct {
	ID               uuid.UUID
	Type             string
	Severity         int32
	Location         Point
	AffectedSegments []int64
	Description      string
	ReportedBy       string
	ReporterID       *uuid.UUID
	Status           string
	AssignedServices []uuid.UUID
	Metadata         map[string]interface{}
	CreatedAt        pgtype.Timestamptz
	UpdatedAt        pgtype.Timestamptz
	ClearedAt        *pgtype.Timestamptz
}

type Alert struct {
	ID          uuid.UUID
	Type        string
	Severity    string
	Title       string
	Message     string
	Geometry    map[string]interface{} // GeoJSON
	TargetRoles []string
	CreatedBy   *uuid.UUID
	ExpiresAt   *pgtype.Timestamptz
	IsActive    bool
	CreatedAt   pgtype.Timestamptz
}