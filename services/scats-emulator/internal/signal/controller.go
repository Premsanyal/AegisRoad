package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/premsanyal/aegisroad/services/scats-emulator/internal/config"
	"github.com/premsanyal/aegisroad/services/scats-emulator/internal/models"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

type Controller struct {
	redis       *redis.Client
	kafka       *kafka.Writer
	intersections map[string]*models.Intersection
	wsClients   map[*WSClient]bool
	wsMu        sync.RWMutex
	intersectionSubs map[string]map[*WSClient]bool
	subMu       sync.RWMutex
	config      *config.Config
}

type WSClient struct {
	send chan []byte
}

func NewController(redisClient *redis.Client, kafkaWriter *kafka.Writer, cfg *config.Config) *Controller {
	return &Controller{
		redis:            redisClient,
		kafka:            kafkaWriter,
		intersections:    make(map[string]*models.Intersection),
		wsClients:        make(map[*WSClient]bool),
		intersectionSubs: make(map[string]map[*WSClient]bool),
		config:           cfg,
	}
}

func (c *Controller) LoadIntersections(configs []config.IntersectionConfig) {
	for _, cfg := range configs {
		intersection := &models.Intersection{
			ID:             cfg.ID,
			Name:           cfg.Name,
			Latitude:       cfg.Latitude,
			Longitude:      cfg.Longitude,
			PhasePlan:      cfg.PhasePlan,
			CurrentPhase:   0,
			PhaseStartTime: time.Now(),
			PhaseEndTime:   time.Now().Add(time.Duration(cfg.PhasePlan.Phases[0].Duration) * time.Second),
			Detectors:      make([]models.Detector, 0),
			ControllerType: cfg.ControllerType,
			IsOnline:       true,
			LastUpdate:     time.Now(),
		}
		c.intersections[cfg.ID] = intersection
		log.Info().Str("id", cfg.ID).Str("name", cfg.Name).Msg("Loaded intersection")
	}
}

func (c *Controller) RunSignalCycles(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.updateSignalCycles()
		}
	}
}

func (c *Controller) updateSignalCycles() {
	now := time.Now()

	for _, intersection := range c.intersections {
		if !intersection.IsOnline {
			continue
		}

		// Check for preemption
		if intersection.PreemptionActive {
			continue // Preemption overrides normal cycle
		}

		// Normal cycle progression
		if now.After(intersection.PhaseEndTime) {
			c.advancePhase(intersection)
		}

		// Update detectors with simulated data
		c.simulateDetectors(intersection)

		// Broadcast status to subscribers
		c.broadcastIntersectionStatus(intersection)
	}
}

func (c *Controller) advancePhase(intersection *models.Intersection) {
	plan := intersection.PhasePlan
	currentIdx := intersection.CurrentPhase

	// Move to next phase
	nextIdx := (currentIdx + 1) % len(plan.Phases)
	nextPhase := plan.Phases[nextIdx]

	intersection.CurrentPhase = nextIdx
	intersection.PhaseStartTime = time.Now()
	intersection.PhaseEndTime = time.Now().Add(time.Duration(nextPhase.Duration) * time.Second)

	log.Debug().
		Str("intersection", intersection.ID).
		Int("phase", nextIdx).
		Int("duration", nextPhase.Duration).
		Msg("Phase advanced")

	// Save to Redis
	c.saveIntersection(intersection)
}

func (c *Controller) simulateDetectors(intersection *models.Intersection) {
	// Simulate realistic detector data based on phase
	currentPhase := intersection.PhasePlan.Phases[intersection.CurrentPhase]

	for i := range intersection.Detectors {
		det := &intersection.Detectors[i]
		// Higher occupancy on green phases for that approach
		isGreen := false
		for _, dir := range currentPhase.Directions {
			if containsDirection(dir, det.Approach) {
				isGreen = true
				break
			}
		}

		if isGreen {
			det.Occupancy = 0.3 + 0.4*float64(time.Now().Unix()%100)/100.0
			det.Volume = 800 + int(400*float64(time.Now().Unix()%100)/100.0)
			det.Speed = 40 + 20*float64(time.Now().Unix()%100)/100.0
		} else {
			det.Occupancy = 0.05 + 0.1*float64(time.Now().Unix()%100)/100.0
			det.Volume = 50 + int(100*float64(time.Now().Unix()%100)/100.0)
			det.Speed = 0
		}
		det.Timestamp = time.Now()
	}
}

func containsDirection(phaseDir, approach string) bool {
	// Simple matching - in production would be more sophisticated
	return phaseDir == approach || phaseDir == "all" || phaseDir == approach+"-left" || phaseDir == approach+"-right"
}

func (c *Controller) broadcastIntersectionStatus(intersection *models.Intersection) {
	status := models.IntersectionStatus{
		IntersectionID: intersection.ID,
		CurrentPhase:   intersection.CurrentPhase,
		PhaseElapsed:   int(time.Since(intersection.PhaseStartTime).Seconds()),
		PhaseRemaining: int(time.Until(intersection.PhaseEndTime).Seconds()),
		NextPhase:      (intersection.CurrentPhase + 1) % len(intersection.PhasePlan.Phases),
		PreemptionActive: intersection.PreemptionActive,
		PreemptionType: intersection.PreemptionType,
		Detectors:      intersection.Detectors,
		Timestamp:      time.Now(),
	}

	data, _ := json.Marshal(models.WSMessage{Type: "intersection_status", Data: status})

	c.subMu.RLock()
	subs := c.intersectionSubs[intersection.ID]
	c.subMu.RUnlock()

	for client := range subs {
		select {
		case client.send <- data:
		default:
		}
	}

	// Also broadcast to "all" subscribers
	c.broadcastToAll(data)
}

func (c *Controller) broadcastToAll(data []byte) {
	c.wsMu.RLock()
	for client := range c.wsClients {
		select {
		case client.send <- data:
		default:
		}
	}
	c.wsMu.RUnlock()
}

// REST API handlers
func (c *Controller) ListIntersections(ctx *gin.Context) {
	result := make([]*models.Intersection, 0, len(c.intersections))
	for _, i := range c.intersections {
		result = append(result, i)
	}
	ctx.JSON(200, result)
}

func (c *Controller) GetIntersection(ctx *gin.Context) {
	id := ctx.Param("id")
	intersection, ok := c.intersections[id]
	if !ok {
		ctx.JSON(404, gin.H{"error": "Intersection not found"})
		return
	}
	ctx.JSON(200, intersection)
}

func (c *Controller) GetIntersectionStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	intersection, ok := c.intersections[id]
	if !ok {
		ctx.JSON(404, gin.H{"error": "Intersection not found"})
		return
	}

	status := models.IntersectionStatus{
		IntersectionID: intersection.ID,
		CurrentPhase:   intersection.CurrentPhase,
		PhaseElapsed:   int(time.Since(intersection.PhaseStartTime).Seconds()),
		PhaseRemaining: int(time.Until(intersection.PhaseEndTime).Seconds()),
		NextPhase:      (intersection.CurrentPhase + 1) % len(intersection.PhasePlan.Phases),
		PreemptionActive: intersection.PreemptionActive,
		PreemptionType: intersection.PreemptionType,
		Detectors:      intersection.Detectors,
		Timestamp:      time.Now(),
	}
	ctx.JSON(200, status)
}

func (c *Controller) SetPhasePlan(ctx *gin.Context) {
	id := ctx.Param("id")
	intersection, ok := c.intersections[id]
	if !ok {
		ctx.JSON(404, gin.H{"error": "Intersection not found"})
		return
	}

	var plan models.PhasePlan
	if err := ctx.ShouldBindJSON(&plan); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if len(plan.Phases) == 0 {
		ctx.JSON(400, gin.H{"error": "At least one phase required"})
		return
	}

	intersection.PhasePlan = plan
	intersection.CurrentPhase = 0
	intersection.PhaseStartTime = time.Now()
	intersection.PhaseEndTime = time.Now().Add(time.Duration(plan.Phases[0].Duration) * time.Second)

	c.saveIntersection(intersection)
	ctx.JSON(200, intersection)
}

func (c *Controller) RequestPreemption(ctx *gin.Context) {
	var req models.PreemptionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	intersection, ok := c.intersections[req.IntersectionID]
	if !ok {
		ctx.JSON(404, gin.H{"error": "Intersection not found"})
		return
	}

	// Check if higher priority preemption is active
	if intersection.PreemptionActive {
		ctx.JSON(409, gin.H{"error": "Preemption already active"})
		return
	}

	// Determine target phase based on approach
	targetPhase := c.findPhaseForApproach(intersection, req.Approach)
	if targetPhase == -1 {
		ctx.JSON(400, gin.H{"error": "No phase found for approach"})
		return
	}

	// Grant preemption
	intersection.PreemptionActive = true
	intersection.PreemptionType = req.VehicleType
	intersection.PreemptionVehicleID = req.VehicleID
	intersection.CurrentPhase = targetPhase
	intersection.PhaseStartTime = time.Now()
	intersection.PhaseEndTime = time.Now().Add(30 * time.Second) // Extended green

	req.Status = "granted"
	now := time.Now()
	req.GrantedAt = &now

	c.saveIntersection(intersection)

	// Publish to Kafka
	c.kafka.WriteMessages(ctx.Request.Context(), kafka.Message{
		Topic: "signal.preemption_granted",
		Value: mustMarshal(req),
	})

	ctx.JSON(200, gin.H{"status": "granted", "phase": targetPhase, "duration": 30})
}

func (c *Controller) CancelPreemption(ctx *gin.Context) {
	preemptionID := ctx.Param("id")

	// Find intersection with this preemption
	for _, intersection := range c.intersections {
		if intersection.PreemptionActive && intersection.PreemptionVehicleID == preemptionID {
			intersection.PreemptionActive = false
			intersection.PreemptionType = ""
			intersection.PreemptionVehicleID = ""
			// Resume normal cycle
			intersection.CurrentPhase = 0
			intersection.PhaseStartTime = time.Now()
			intersection.PhaseEndTime = time.Now().Add(time.Duration(intersection.PhasePlan.Phases[0].Duration) * time.Second)

			c.saveIntersection(intersection)

			ctx.JSON(200, gin.H{"status": "cancelled"})
			return
		}
	}

	ctx.JSON(404, gin.H{"error": "Preemption not found"})
}

func (c *Controller) ListActivePreemptions(ctx *gin.Context) {
	var active []models.Intersection
	for _, intersection := range c.intersections {
		if intersection.PreemptionActive {
			active = append(active, *intersection)
		}
	}
	ctx.JSON(200, active)
}

func (c *Controller) GetDetectors(ctx *gin.Context) {
	id := ctx.Param("id")
	intersection, ok := c.intersections[id]
	if !ok {
		ctx.JSON(404, gin.H{"error": "Intersection not found"})
		return
	}
	ctx.JSON(200, intersection.Detectors)
}

func (c *Controller) UpdateDetector(ctx *gin.Context) {
	id := ctx.Param("id")
	intersection, ok := c.intersections[id]
	if !ok {
		ctx.JSON(404, gin.H{"error": "Intersection not found"})
		return
	}

	var det models.Detector
	if err := ctx.ShouldBindJSON(&det); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Update or add detector
	found := false
	for i := range intersection.Detectors {
		if intersection.Detectors[i].ID == det.ID {
			intersection.Detectors[i] = det
			found = true
			break
		}
	}
	if !found {
		intersection.Detectors = append(intersection.Detectors, det)
	}

	c.saveIntersection(intersection)
	ctx.JSON(200, det)
}

// WebSocket management
func (c *Controller) RegisterWSClient(client *WSClient) {
	c.wsMu.Lock()
	c.wsClients[client] = true
	c.wsMu.Unlock()
}

func (c *Controller) UnregisterWSClient(client *WSClient) {
	c.wsMu.Lock()
	delete(c.wsClients, client)
	c.wsMu.Unlock()

	c.subMu.Lock()
	for _, subs := range c.intersectionSubs {
		delete(subs, client)
	}
	c.subMu.Unlock()
}

func (c *Controller) SubscribeIntersection(client *WSClient, intersectionID string) {
	c.subMu.Lock()
	if c.intersectionSubs[intersectionID] == nil {
		c.intersectionSubs[intersectionID] = make(map[*WSClient]bool)
	}
	c.intersectionSubs[intersectionID][client] = true
	c.subMu.Unlock()
}

func (c *Controller) UnsubscribeIntersection(client *WSClient, intersectionID string) {
	c.subMu.Lock()
	delete(c.intersectionSubs[intersectionID], client)
	c.subMu.Unlock()
}

func (c *Controller) findPhaseForApproach(intersection *models.Intersection, approach string) int {
	for i, phase := range intersection.PhasePlan.Phases {
		for _, dir := range phase.Directions {
			if containsDirection(dir, approach) {
				return i
			}
		}
	}
	return -1
}

func (c *Controller) saveIntersection(intersection *models.Intersection) {
	data, _ := json.Marshal(intersection)
	c.redis.Set(c.redis.Context(), "intersection:"+intersection.ID, data, 0)
}

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}