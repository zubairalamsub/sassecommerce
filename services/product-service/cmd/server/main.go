package main

import (
	"context"
	"errors"
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
	sharedconfig "github.com/ecommerce/shared/go/pkg/config"
	"github.com/ecommerce/shared/go/pkg/metrics"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	sharedstorage "github.com/ecommerce/shared/go/pkg/storage"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
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
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			logger.WithError(err).Error("Error closing Kafka producer")
		}
	}()

	// Optional Redis cache for the product read path. If REDIS_HOST is unset
	// the service runs without caching (repo is passed a nil client).
	var redisClient *redis.Client
	if redisHost := getEnv("REDIS_HOST", ""); redisHost != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     redisHost + ":" + getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
		})
		pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
		if err := redisClient.Ping(pingCtx).Err(); err != nil {
			logger.WithError(err).Warn("Redis unavailable; continuing without product cache")
			redisClient = nil
		} else {
			logger.Info("Product cache enabled (Redis)")
		}
		cancelPing()
		defer func() {
			if redisClient != nil {
				_ = redisClient.Close()
			}
		}()
	}

	// Initialize repositories
	productRepo := repository.NewProductRepository(db, redisClient)
	categoryRepo := repository.NewCategoryRepository(db)

	// Ensure MongoDB indexes exist for the hot read paths (idempotent).
	idxCtx, cancelIdx := context.WithTimeout(context.Background(), 30*time.Second)
	if err := productRepo.EnsureIndexes(idxCtx); err != nil {
		logger.WithError(err).Warn("Failed to ensure product indexes; reads may be slow")
	}
	cancelIdx()

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

	// Optional: initialise the object storage client and image handlers.
	// Only register image upload routes when storage is configured — services
	// running without OCI credentials (e.g. local dev without S3) should still
	// boot cleanly with category/product CRUD intact.
	var imageHandler *api.ImageHandler
	var categoryImageHandler *api.CategoryImageHandler
	if storageCfg, err := sharedstorage.NewFromEnv(os.Getenv); err == nil {
		storageClient, err := sharedstorage.New(context.Background(), *storageCfg)
		if err != nil {
			logger.WithError(err).Warn("Object storage configured but client init failed; image uploads disabled")
		} else {
			imageService := service.NewImageService(productRepo, storageClient, logger)
			imageHandler = api.NewImageHandler(imageService, logger)

			categoryImageService := service.NewCategoryImageService(categoryRepo, storageClient, logger)
			categoryImageHandler = api.NewCategoryImageHandler(categoryImageService, logger)

			logger.WithField("bucket", storageClient.Bucket()).Info("Object storage configured; image upload routes enabled")
		}
	} else {
		logger.WithError(err).Info("Object storage not configured; image upload routes disabled")
	}

	// Setup router
	router := setupRouter(config, logger, productHandler, categoryHandler, imageHandler, categoryImageHandler)

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", config.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("port", config.Port).Info("Starting Product Service")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	Port      string
	MongoURI  string
	DBName    string
	JWTSecret string
}

func loadConfig() *Config {
	return &Config{
		Port:      getEnv("PORT", "8083"),
		MongoURI:  getEnv("MONGO_URI", "mongodb://mongodb:27017"),
		DBName:    getEnv("DB_NAME", "product_db"),
		JWTSecret: sharedconfig.MustGetJWTSecret(),
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

func setupRouter(config *Config, logger *logrus.Logger, productHandler *api.ProductHandler, categoryHandler *api.CategoryHandler, imageHandler *api.ImageHandler, categoryImageHandler *api.CategoryImageHandler) *gin.Engine {
	// Set Gin mode based on environment
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())
	// Security headers land early so they apply to every response — including
	// errors raised by middleware further down the chain.
	router.Use(sharedmiddleware.SecurityHeaders(sharedmiddleware.SecurityHeadersConfig{}))
	router.Use(metrics.Middleware("product-service"))
	router.Use(sharedmiddleware.HardenedCORS(sharedmiddleware.CORSConfig{
		AllowCredentials: true,
	}))
	router.Use(sharedmiddleware.RequestLogger(sharedmiddleware.RequestLoggerConfig{
		Logger:          logger,
		LogRequestBody:  true,
		LogResponseBody: true,
		SkipPaths:       []string{"/health", "/ready", "/metrics"},
	}))

	// Health check endpoint (no authentication required)
	router.GET("/health", healthCheck)
	router.GET("/ready", readinessCheck)

	// Prometheus metrics endpoint — registered before Auth/RateLimit so scrapes are not blocked.
	router.GET("/metrics", sharedmiddleware.MetricsAuth(), gin.WrapH(metrics.Handler()))

	// Rate limiting (after /metrics so Prometheus scrapes aren't throttled)
	router.Use(sharedmiddleware.RateLimit(sharedmiddleware.RateLimitConfig{
		Rate:   100,
		Window: time.Minute,
	}))

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

		// Image upload routes are admin/moderator-only — they live under their
		// own groups so we can apply auth + role middleware uniformly without
		// touching the public read routes registered above.
		if imageHandler != nil {
			productImages := v1.Group("/products")
			productImages.Use(authMw)
			productImages.Use(sharedmiddleware.RequireRole("admin", "moderator"))
			imageHandler.RegisterRoutes(productImages)
		}
		if categoryImageHandler != nil {
			categoryImages := v1.Group("/categories")
			categoryImages.Use(authMw)
			categoryImages.Use(sharedmiddleware.RequireRole("admin", "moderator"))
			categoryImageHandler.RegisterRoutes(categoryImages)
		}
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
