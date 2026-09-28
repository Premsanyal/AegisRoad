package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/premsanyal/aegisroad/services/scats-emulator/internal/config"
	"github.com/premsanyal/aegisroad/services/scats-emulator/internal/models"
	"github.com/premsanyal/aegisroad/services/scats-emulator/internal/signal"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	setupLogger(cfg.LogLevel)

	log.Info().Msg("Starting SCATS Emulator")

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	defer redisClient.Close()

	// Initialize Kafka
	kafkaWriter := &kafka.Writer{
		Addr:     kafka.TCP(cfg.KafkaBrokers),
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close()

	// Initialize signal controller
	controller := signal.NewController(redisClient, kafkaWriter, cfg)
	controller.LoadIntersections(cfg.IntersectionConfigs)

	// Start signal cycle simulation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go controller.RunSignalCycles(ctx)

	// Setup HTTP server
	router := setupRouter(cfg, controller, redisClient)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Info().Msg("Shutting down...")
		cancel()
		server.Close()
	}()

	log.Info().Int("port", cfg.HTTPPort).Msg("SCATS Emulator starting")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server failed")
	}
}

func setupLogger(level string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
}

func setupRouter(cfg *config.Config, controller *signal.Controller, redisClient *redis.Client) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(loggingMiddleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "scats-emulator"})
	})

	// REST API
	api := r.Group("/scats/v1")
	{
		// Intersection status
		api.GET("/intersections", controller.ListIntersections)
		api.GET("/intersections/:id", controller.GetIntersection)
		api.GET("/intersections/:id/status", controller.GetIntersectionStatus)
		api.POST("/intersections/:id/phase-plan", controller.SetPhasePlan)

		// Preemption
		api.POST("/preemption/request", controller.RequestPreemption)
		api.POST("/preemption/:id/cancel", controller.CancelPreemption)
		api.GET("/preemption/active", controller.ListActivePreemptions)

		// Detectors
		api.GET("/intersections/:id/detectors", controller.GetDetectors)
		api.POST("/intersections/:id/detectors", controller.UpdateDetector)
	}

	// WebSocket for real-time updates
	r.GET("/scats/live", func(c *gin.Context) {
		handleWebSocket(c.Writer, c.Request, controller)
	})

	return r
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", time.Since(start)).
			Msg("HTTP request")
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, controller *signal.Controller) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("WebSocket upgrade failed")
		return
	}
	defer conn.Close()

	client := &WSClient{conn: conn, controller: controller, send: make(chan []byte, 256)}
	controller.RegisterWSClient(client)
	defer controller.UnregisterWSClient(client)

	go client.writePump()
	client.readPump()
}

type WSClient struct {
	conn       *websocket.Conn
	controller *signal.Controller
	send       chan []byte
	mu         sync.Mutex
}

func (c *WSClient) readPump() {
	defer func() {
		c.controller.UnregisterWSClient(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg models.WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "subscribe_intersection":
			if id, ok := msg.Data.(string); ok {
				c.controller.SubscribeIntersection(c, id)
			}
		case "unsubscribe_intersection":
			if id, ok := msg.Data.(string); ok {
				c.controller.UnsubscribeIntersection(c, id)
			}
		case "ping":
			c.send <- []byte(`{"type":"pong"}`)
		}
	}
}

func (c *WSClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}