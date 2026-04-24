package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"strings"

	"github.com/ecommerce/product-service/internal/api"
	"github.com/ecommerce/product-service/internal/messaging"
	"github.com/ecommerce/product-service/internal/repository"
	"github.com/ecommerce/product-service/internal/service"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Load configuration from environment
	config := loadConfig()

	// Connect to MongoDB
	client, err := connectMongoDB(config.MongoURI, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to MongoDB")
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			logger.WithError(err).Error("Error disconnecting from MongoDB")
		}
	}()

	db := client.Database(config.DBName)
	logger.Info("Successfully connected to MongoDB")

	// Initialize Kafka producer
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ",")
	kafkaProducer := messaging.NewProducer(kafkaBrokers, logger)
	defer kafkaProducer.Close()

	// Initialize repositories
	productRepo := repository.NewProductRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)

	// Initialize Kafka consumer for inventory events
	kafkaConsumer := messaging.NewEventConsumer(kafkaBrokers, "product-service", productRepo, logger)
	ctx, cancelConsumer := context.WithCancel(context.Background())
	kafkaConsumer.Start(ctx)
	defer func() {
		cancelConsumer()
		kafkaConsumer.Stop()
	}()

	// Initialize services
	productService := service.NewProductService(productRepo, categoryRepo, kafkaProducer, logger)
	categoryService := service.NewCategoryService(categoryRepo, logger)

	// Initialize handlers
	productHandler := api.NewProductHandler(productService, logger)
	categoryHandler := api.NewCategoryHandler(categoryService, logger)

	// Setup router
	router := setupRouter(config, logger, productHandler, categoryHandler)

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("port", config.Port).Info("Starting Product Service")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.WithError(err).Fatal("Server forced to shutdown")
	}

	logger.Info("Server exited")
}

// Config holds the application configuration
type Config struct {
	Port     string
	MongoURI string
	DBName   string
	JWTSecret string
}

func loadConfig() *Config {
	return &Config{
		Port:     getEnv("PORT", "8083"),
		MongoURI: getEnv("MONGO_URI", "mongodb://mongodb:27017"),
		DBName:   getEnv("DB_NAME", "product_db"),
		JWTSecret: getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func connectMongoDB(uri string, logger *logrus.Logger) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return client, nil
}

func setupRouter(config *Config, logger *logrus.Logger, productHandler *api.ProductHandler, categoryHandler *api.CategoryHandler) *gin.Engine {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(requestLogger(logger))
	router.Use(sharedmiddleware.RateLimit(sharedmiddleware.RateLimitConfig{
		Rate:   100,
		Window: time.Minute,
	}))

	// Health check endpoint (no authentication required)
	router.GET("/health", healthCheck)
	router.GET("/ready", readinessCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Apply tenant middleware to all v1 routes
		v1.Use(sharedmiddleware.Tenant(sharedmiddleware.TenantConfig{
			Required:    true,
			AllowHeader: true,
		}))

		// Auth middleware for protected write routes
		authMw := sharedmiddleware.Auth(sharedmiddleware.AuthConfig{
			SecretKey: config.JWTSecret,
		})

		// Register route handlers (auth middleware applied to write routes)
		productHandler.RegisterRoutes(v1, authMw)
		categoryHandler.RegisterRoutes(v1, authMw)
	}

	return router
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "product-service",
		"time":    time.Now().UTC(),
	})
}

func readinessCheck(c *gin.Context) {
	// TODO: Add database connectivity check
	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"service": "product-service",
	})
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Tenant-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func requestLogger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Log request details
		duration := time.Since(startTime)
		logger.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"duration":   duration.Milliseconds(),
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"tenant_id":  c.GetString("tenant_id"),
		}).Info("Request processed")
	}
}
