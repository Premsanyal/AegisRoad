package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/auth"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/config"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/handlers"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/middleware"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/routing"
	"github.com/premsanyal/aegisroad/services/api-gateway/internal/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	// Setup logger
	setupLogger(cfg.LogLevel)

	log.Info().Msg("Starting AegisRoad API Gateway")

	// Initialize database connection
	db, err := initDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize Redis
	redisClient, err := initRedis(cfg.RedisURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer redisClient.Close()

	// Initialize Kafka producer
	kafkaProducer := initKafka(cfg.KafkaBrokers)
	defer kafkaProducer.Close()

	// Initialize gRPC client for routing engine
	routingClient, err := routing.NewClient(cfg.RoutingGRPCAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to routing engine")
	}
	defer routingClient.Close()

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(redisClient)
	go wsHub.Run()

	// Initialize handlers
	routeHandler := handlers.NewRouteHandler(routingClient, kafkaProducer, db)
	incidentHandler := handlers.NewIncidentHandler(db, kafkaProducer, wsHub)
	vehicleHandler := handlers.NewVehicleHandler(db, kafkaProducer, wsHub)
	authHandler := handlers.NewAuthHandler(db, cfg.JWTSecret)
	alertHandler := handlers.NewAlertHandler(db, kafkaProducer, wsHub)

	// Setup Gin router
	router := setupRouter(cfg, routeHandler, incidentHandler, vehicleHandler, authHandler, alertHandler, wsHub)

	// Start HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: router,
	}

	// Start gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.GRPCAuthInterceptor(cfg.JWTSecret)),
	)
	// Register gRPC services here

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Info().Msg("Shutting down servers...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("HTTP server shutdown error")
		}

		grpcServer.GracefulStop()
		wsHub.Close()
	}()

	// Start HTTP server
	go func() {
		log.Info().Int("port", cfg.HTTPPort).Msg("HTTP server starting")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to listen for gRPC")
		}
		log.Info().Int("port", cfg.GRPCPort).Msg("gRPC server starting")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("gRPC server failed")
		}
	}()

	// Wait for shutdown
	select {}
}

func setupLogger(level string) {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
}

func initDB(databaseURL string) (*sqlx.DB, error) {
	// Implementation would use sqlx with pgx driver
	return nil, nil // Placeholder
}

func initRedis(redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return client, nil
}

func initKafka(brokers string) *kafka.Writer {
	return &kafka.Writer{
		Addr:     kafka.TCP(brokers),
		Balancer: &kafka.LeastBytes{},
	}
}

func setupRouter(
	cfg *config.Config,
	routeHandler *handlers.RouteHandler,
	incidentHandler *handlers.IncidentHandler,
	vehicleHandler *handlers.VehicleHandler,
	authHandler *handlers.AuthHandler,
	alertHandler *handlers.AlertHandler,
	wsHub *websocket.Hub,
) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "api-gateway"})
	})

	// Public routes (no auth)
	public := r.Group("/api/v1")
	{
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/register", authHandler.Register)
		public.POST("/auth/refresh", authHandler.RefreshToken)
	}

	// Protected routes
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		// Routes
		protected.POST("/routes", routeHandler.CreateRoute)
		protected.GET("/routes/:id", routeHandler.GetRoute)
		protected.GET("/routes", routeHandler.ListRoutes)
		protected.POST("/routes/:id/accept", routeHandler.AcceptRoute)
		protected.POST("/routes/:id/cancel", routeHandler.CancelRoute)

		// Incidents
		protected.POST("/incidents", incidentHandler.CreateIncident)
		protected.GET("/incidents/:id", incidentHandler.GetIncident)
		protected.GET("/incidents", incidentHandler.ListIncidents)
		protected.PUT("/incidents/:id", incidentHandler.UpdateIncident)
		protected.POST("/incidents/:id/verify", incidentHandler.VerifyIncident)
		protected.POST("/incidents/:id/assign", incidentHandler.AssignServices)

		// Vehicles
		protected.GET("/vehicles/:id", vehicleHandler.GetVehicle)
		protected.PUT("/vehicles/:id/location", vehicleHandler.UpdateLocation)
		protected.PUT("/vehicles/:id/status", vehicleHandler.UpdateStatus)

		// Alerts (operator only)
		protected.POST("/alerts", middleware.RequireRole("operator", "admin"), alertHandler.CreateAlert)
		protected.GET("/alerts", alertHandler.ListAlerts)
	}

	// WebSocket endpoint
	r.GET("/ws/v1/live", wsHub.HandleWebSocket)

	return r
}