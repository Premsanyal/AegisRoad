package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/auth"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/models"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/routing"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/websocket"
	"github.com/segmentio/kafka-go"
	"github.com/rs/zerolog/log"
)

type RouteHandler struct {
	routingClient *routing.Client
	kafkaWriter   *kafka.Writer
	db            interface{} // *sqlx.DB
}

func NewRouteHandler(rc *routing.Client, kw *kafka.Writer, db interface{}) *RouteHandler {
	return &RouteHandler{
		routingClient: rc,
		kafkaWriter:   kw,
		db:            db,
	}
}

func (h *RouteHandler) CreateRoute(c *gin.Context) {
	var req struct {
		Origin          models.Point `json:"origin" binding:"required"`
		Destination     models.Point `json:"destination" binding:"required"`
		VehicleType     string       `json:"vehicle_type"`
		PriorityLevel   int32        `json:"priority_level"`
		ReturnAlternatives bool      `json:"return_alternatives"`
		MaxAlternatives int          `json:"max_alternatives"`
		Profile         string       `json:"profile"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	userRole := c.GetString("user_role")

	// Determine priority based on role
	priorityLevel := req.PriorityLevel
	if userRole == "emergency" && priorityLevel == 0 {
		priorityLevel = 2 // Emergency gets priority 2
	} else if userRole == "operator" && priorityLevel == 0 {
		priorityLevel = 1 // Operators get priority 1
	}

	routeReq := &models.RouteRequest{
		RequestID:          uuid.New().String(),
		PriorityLevel:      priorityLevel,
		Origin:             req.Origin,
		Destination:        req.Destination,
		VehicleID:          c.GetString("vehicle_id"),
		VehicleType:        req.VehicleType,
		ReturnAlternatives: req.ReturnAlternatives,
		MaxAlternatives:    req.MaxAlternatives,
		Profile:            parseProfile(req.Profile),
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	resp, err := h.routingClient.CalculateRoute(ctx, routeReq)
	if err != nil {
		log.Error().Err(err).Str("request_id", routeReq.RequestID).Msg("Route calculation failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Route calculation failed"})
		return
	}

	// For public users, return only alternative routes (not the best)
	if userRole == "public" && len(resp.AlternativeRoutes) > 0 {
		// The primary route is reserved for emergency - offer 2nd best as "recommended"
		c.JSON(http.StatusOK, gin.H{
			"request_id":         resp.RequestID,
			"recommended_route":  resp.AlternativeRoutes[0], // 2nd best
			"alternative_routes": resp.AlternativeRoutes[1:], // 3rd, 4th best
			"status":             resp.Status,
			"calculated_at":      resp.CalculatedAt,
		})
		return
	}

	// Emergency/operators get full response
	c.JSON(http.StatusOK, resp)
}

func (h *RouteHandler) GetRoute(c *gin.Context) {
	routeID := c.Param("id")
	// Query database for route
	c.JSON(http.StatusOK, gin.H{"route_id": routeID, "message": "Not implemented"})
}

func (h *RouteHandler) ListRoutes(c *gin.Context) {
	userID := c.GetString("user_id")
	// Query database for user's routes
	c.JSON(http.StatusOK, gin.H{"routes": []interface{}{}, "message": "Not implemented"})
}

func (h *RouteHandler) AcceptRoute(c *gin.Context) {
	routeID := c.Param("id")
	userID := c.GetString("user_id")

	// Update route status in database
	// Publish to Kafka for real-time updates

	c.JSON(http.StatusOK, gin.H{"route_id": routeID, "status": "accepted", "accepted_by": userID})
}

func (h *RouteHandler) CancelRoute(c *gin.Context) {
	routeID := c.Param("id")
	// Update route status in database
	c.JSON(http.StatusOK, gin.H{"route_id": routeID, "status": "cancelled"})
}

func parseProfile(profile string) models.RoutingProfile {
	switch profile {
	case "bike":
		return models.ProfileBike
	case "foot":
		return models.ProfileFoot
	case "emergency":
		return models.ProfileEmergency
	default:
		return models.ProfileCar
	}
}

// AuthHandler handles authentication
type AuthHandler struct {
	db       interface{}
	jwtSecret string
}

func NewAuthHandler(db interface{}, jwtSecret string) *AuthHandler {
	return &AuthHandler{db: db, jwtSecret: jwtSecret}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In production: verify phone/password against database
	// For now, mock response
	userID := uuid.New().String()
	role := "public"

	accessToken, refreshToken, err := auth.GenerateTokenPair(userID, role, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":   userID,
			"role": role,
		},
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone" binding:"required"`
		Password string `json:"password" binding:"required"`
		Name     string `json:"name"`
		Role     string `json:"role"` // public, emergency, operator
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In production: create user in database
	userID := uuid.New().String()
	role := req.Role
	if role == "" {
		role = "public"
	}

	accessToken, refreshToken, err := auth.GenerateTokenPair(userID, role, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id":   userID,
			"role": role,
		},
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claims, err := auth.ValidateToken(req.RefreshToken, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	accessToken, refreshToken, err := auth.GenerateTokenPair(claims.UserID, claims.Role, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}