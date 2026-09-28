package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/models"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/websocket"
	"github.com/segmentio/kafka-go"
	"github.com/rs/zerolog/log"
)

type IncidentHandler struct {
	db         interface{}
	kafkaWriter *kafka.Writer
	wsHub      *websocket.Hub
}

func NewIncidentHandler(db interface{}, kw *kafka.Writer, wsHub *websocket.Hub) *IncidentHandler {
	return &IncidentHandler{
		db:          db,
		kafkaWriter: kw,
		wsHub:       wsHub,
	}
}

func (h *IncidentHandler) CreateIncident(c *gin.Context) {
	var req struct {
		Type             string   `json:"type" binding:"required"`
		Severity         int32    `json:"severity" binding:"required,min=1,max=5"`
		Location         models.Point `json:"location" binding:"required"`
		Description      string   `json:"description"`
		AffectedSegments []int64  `json:"affected_segments"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	incident := &models.Incident{
		ID:               uuid.New(),
		Type:             req.Type,
		Severity:         req.Severity,
		Location:         req.Location,
		AffectedSegments: req.AffectedSegments,
		Description:      req.Description,
		ReportedBy:       userRole,
		ReporterID:       parseUUID(userID),
		Status:           "active",
		Metadata:         map[string]interface{}{},
	}

	// In production: save to database
	// In production: publish to Kafka for real-time updates
	// In production: broadcast to WebSocket clients

	// Broadcast to relevant clients
	h.wsHub.BroadcastToRole("operator", websocket.Message{
		Type: "incident_created",
		Data: incident,
	})

	if req.Severity >= 3 {
		h.wsHub.BroadcastToRole("emergency", websocket.Message{
			Type: "incident_high_severity",
			Data: incident,
		})
	}

	c.JSON(http.StatusCreated, incident)
}

func (h *IncidentHandler) GetIncident(c *gin.Context) {
	incidentID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"incident_id": incidentID, "message": "Not implemented"})
}

func (h *IncidentHandler) ListIncidents(c *gin.Context) {
	status := c.DefaultQuery("status", "active")
	// Query database
	c.JSON(http.StatusOK, gin.H{"status": status, "incidents": []interface{}{}})
}

func (h *IncidentHandler) UpdateIncident(c *gin.Context) {
	incidentID := c.Param("id")
	var req struct {
		Status       string `json:"status"`
		Severity     int32  `json:"severity"`
		Description  string `json:"description"`
	}
	c.ShouldBindJSON(&req)
	c.JSON(http.StatusOK, gin.H{"incident_id": incidentID, "updated": req})
}

func (h *IncidentHandler) VerifyIncident(c *gin.Context) {
	incidentID := c.Param("id")
	// Update incident status to "verified" or "active"
	// Broadcast update
	h.wsHub.BroadcastToRole("operator", websocket.Message{
		Type: "incident_verified",
		Data: gin.H{"incident_id": incidentID},
	})
	c.JSON(http.StatusOK, gin.H{"incident_id": incidentID, "status": "verified"})
}

func (h *IncidentHandler) AssignServices(c *gin.Context) {
	incidentID := c.Param("id")
	var req struct {
		VehicleIDs []string `json:"vehicle_ids" binding:"required"`
	}
	c.ShouldBindJSON(&req)

	// Assign vehicles to incident
	// Broadcast dispatch notification
	for _, vid := range req.VehicleIDs {
		h.wsHub.BroadcastToVehicle(vid, websocket.Message{
			Type: "dispatch",
			Data: gin.H{
				"incident_id": incidentID,
				"action":      "respond",
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{"incident_id": incidentID, "assigned": req.VehicleIDs})
}

func parseUUID(s string) *uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}