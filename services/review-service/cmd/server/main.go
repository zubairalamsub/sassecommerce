package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ecommerce/review-service/internal/api"
	"github.com/ecommerce/review-service/internal/config"
	"github.com/ecommerce/review-service/internal/messaging"
	"github.com/ecommerce/review-service/internal/repository"
	"github.com/ecommerce/review-service/internal/service"
	"github.com/ecommerce/review-service/pkg/logger"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg := config.Load()
	log := logger.NewLogger(cfg.Server.Env)
	log.Info("Starting Review & Rating Service...")

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoDB.URI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Info("Successfully connected to MongoDB")

	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.WithError(err).Error("Failed to disconnect from MongoDB")
		}
	}()

	db := mongoClient.Database(cfg.MongoDB.DBName)

	// Initialize Kafka writer
	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Kafka.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
	}
	defer kafkaWriter.Close()

	// Initialize repository
	reviewRepo := repository.NewReviewRepository(db)

	// Initialize service
	reviewService := service.NewReviewService(reviewRepo, kafkaWriter, log)

	// Initialize Kafka consumer
	consumer := messaging.NewEventConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, log)
	consumer.Start(context.Background())
	defer consumer.Stop()

	// Initialize handler
	handler := api.NewReviewHandler(reviewService, log)

	// Setup Gin router
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Tenant-Id"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "review-service",
			"time":    time.Now().UTC(),
		})
	})

	// JWT Auth middleware
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production-12345"
	}
	router.Use(sharedmiddleware.Auth(sharedmiddleware.AuthConfig{SecretKey: jwtSecret}))

	api.RegisterRoutes(router, handler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Infof("Review Service listening on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Server exited")
}
