package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/models"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/websocket"
	"github.com/segmentio/kafka-go"
	"github.com/rs/zerolog/log"
)

type VehicleHandler struct {
	db          interface{}
	kafkaWriter *kafka.Writer
	wsHub       *websocket.Hub
}

func NewVehicleHandler(db interface{}, kw *kafka.Writer, wsHub *websocket.Hub) *VehicleHandler {
	return &VehicleHandler{
		db:          db,
		kafkaWriter: kw,
		wsHub:       wsHub,
	}
}

func (h *VehicleHandler) GetVehicle(c *gin.Context) {
	vehicleID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"vehicle_id": vehicleID, "message": "Not implemented"})
}

func (h *VehicleHandler) UpdateLocation(c *gin.Context) {
	vehicleID := c.Param("id")
	var req struct {
		Location models.Point `json:"location" binding:"required"`
		Heading  float64      `json:"heading"`
		SpeedKmh float64      `json:"speed_kmh"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update vehicle location in database
	// Publish to Kafka for real-time tracking
	// Broadcast to WebSocket clients tracking this vehicle

	h.wsHub.BroadcastToVehicle(vehicleID, websocket.Message{
		Type: "location_update",
		Data: gin.H{
			"vehicle_id": vehicleID,
			"location":   req.Location,
			"heading":    req.Heading,
			"speed_kmh":  req.SpeedKmh,
		},
	})

	// If emergency vehicle, update corridor reservation
	// (handled by routing engine via Kafka)

	c.JSON(http.StatusOK, gin.H{"vehicle_id": vehicleID, "location": req.Location})
}

func (h *VehicleHandler) UpdateStatus(c *gin.Context) {
	vehicleID := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"` // idle, enroute, on_scene, returning
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update status in database
	// If status changed to "on_scene", notify incident
	// If status changed to "idle", release corridor reservation

	h.wsHub.BroadcastToRole("operator", websocket.Message{
		Type: "vehicle_status",
		Data: gin.H{
			"vehicle_id": vehicleID,
			"status":     req.Status,
		},
	})

	c.JSON(http.StatusOK, gin.H{"vehicle_id": vehicleID, "status": req.Status})
}

// AlertHandler handles broadcast alerts
type AlertHandler struct {
	db          interface{}
	kafkaWriter *kafka.Writer
	wsHub       *websocket.Hub
}

func NewAlertHandler(db interface{}, kw *kafka.Writer, wsHub *websocket.Hub) *AlertHandler {
	return &AlertHandler{
		db:          db,
		kafkaWriter: kw,
		wsHub:       wsHub,
	}
}

func (h *AlertHandler) CreateAlert(c *gin.Context) {
	var req struct {
		Type        string                 `json:"type" binding:"required"`
		Severity    string                 `json:"severity" binding:"required"`
		Title       string                 `json:"title" binding:"required"`
		Message     string                 `json:"message" binding:"required"`
		Geometry    map[string]interface{} `json:"geometry"` // GeoJSON polygon
		TargetRoles []string               `json:"target_roles"`
		ExpiresIn   int                    `json:"expires_in_minutes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")

	alert := &models.Alert{
		ID:          uuid.New(),
		Type:        req.Type,
		Severity:    req.Severity,
		Title:       req.Title,
		Message:     req.Message,
		Geometry:    req.Geometry,
		TargetRoles: req.TargetRoles,
		CreatedBy:   parseUUID(userID),
		IsActive:    true,
	}

	if req.ExpiresIn > 0 {
		// Set expiration
	}

	// Save to database
	// Publish to Kafka
	// Broadcast via WebSocket

	for _, role := range req.TargetRoles {
		h.wsHub.BroadcastToRole(role, websocket.Message{
			Type: "alert",
			Data: alert,
		})
	}

	// If no target roles specified, broadcast to all
	if len(req.TargetRoles) == 0 {
		h.wsHub.BroadcastAll(websocket.Message{
			Type: "alert",
			Data: alert,
		})
	}

	c.JSON(http.StatusCreated, alert)
}

func (h *AlertHandler) ListAlerts(c *gin.Context) {
	// Query active alerts
	c.JSON(http.StatusOK, gin.H{"alerts": []interface{}{}})
}

func parseUUID(s string) *uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}