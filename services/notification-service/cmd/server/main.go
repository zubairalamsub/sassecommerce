package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ecommerce/notification-service/internal/api"
	"github.com/ecommerce/notification-service/internal/config"
	"github.com/ecommerce/notification-service/internal/messaging"
	"github.com/ecommerce/notification-service/internal/models"
	"github.com/ecommerce/notification-service/internal/repository"
	"github.com/ecommerce/notification-service/internal/service"
	"github.com/ecommerce/notification-service/pkg/logger"
	sharedconfig "github.com/ecommerce/shared/go/pkg/config"
	"github.com/ecommerce/shared/go/pkg/metrics"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.NewLogger(cfg.Server.Env)
	log.Info("Starting Notification Service...")

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

	// Initialize repository
	notifRepo := repository.NewNotificationRepository(db)

	// Initialize notification providers — use real providers when API keys are
	// configured, falling back to simulated providers for development.
	providers := make(map[models.Channel]service.NotificationProvider)

	if sgKey := os.Getenv("SENDGRID_API_KEY"); sgKey != "" {
		providers[models.ChannelEmail] = service.NewSendGridEmailProvider(service.SendGridConfig{
			APIKey:    sgKey,
			FromEmail: getEnv("SENDGRID_FROM_EMAIL", "noreply@saajan.com"),
			FromName:  getEnv("SENDGRID_FROM_NAME", "Saajan Store"),
		}, log)
		log.Info("Email provider: SendGrid")
	} else {
		providers[models.ChannelEmail] = service.NewSimulatedEmailProvider(log)
		log.Info("Email provider: Simulated (set SENDGRID_API_KEY to enable SendGrid)")
	}

	if twilioSID := os.Getenv("TWILIO_ACCOUNT_SID"); twilioSID != "" {
		providers[models.ChannelSMS] = service.NewTwilioSMSProvider(service.TwilioConfig{
			AccountSID: twilioSID,
			AuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
			FromNumber: getEnv("TWILIO_FROM_NUMBER", "+15005550006"),
		}, log)
		log.Info("SMS provider: Twilio")
	} else {
		providers[models.ChannelSMS] = service.NewSimulatedSMSProvider(log)
		log.Info("SMS provider: Simulated (set TWILIO_ACCOUNT_SID to enable Twilio)")
	}

	if fcmKey := os.Getenv("FCM_SERVER_KEY"); fcmKey != "" {
		providers[models.ChannelPush] = service.NewFCMPushProvider(service.FCMConfig{
			ServerKey: fcmKey,
			ProjectID: os.Getenv("FCM_PROJECT_ID"),
		}, log)
		log.Info("Push provider: Firebase Cloud Messaging")
	} else {
		providers[models.ChannelPush] = service.NewSimulatedPushProvider(log)
		log.Info("Push provider: Simulated (set FCM_SERVER_KEY to enable FCM)")
	}

	// Initialize service
	notifService := service.NewNotificationService(notifRepo, providers, log)
	tmplService := service.NewTemplateService(notifRepo, providers, log)

	// Initialize Kafka consumer — templates are looked up via the repository
	// directly so the consumer can override the hardcoded RenderEmailHTML
	// output when a tenant has an active template configured.
	frontendBaseURL := getEnv("FRONTEND_BASE_URL", "https://shop.example.com")
	consumer := messaging.NewEventConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, notifService, notifRepo, frontendBaseURL, log)
	consumer.Start(context.Background())
	defer consumer.Stop()

	// Initialize handler
	handler := api.NewNotificationHandler(notifService, log)
	tmplHandler := api.NewTemplateHandler(tmplService, log)

	// Setup Gin router
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	// Security headers land early so they apply to every response — including
	// errors raised by middleware further down the chain.
	router.Use(sharedmiddleware.SecurityHeaders(sharedmiddleware.SecurityHeadersConfig{}))
	router.Use(metrics.Middleware("notification-service"))
	router.Use(sharedmiddleware.RequestLogger(sharedmiddleware.RequestLoggerConfig{
		Logger:          log,
		LogRequestBody:  true,
		LogResponseBody: true,
		SkipPaths:       []string{"/health", "/ready", "/metrics"},
	}))

	// CORS: origins are sourced from CORS_ALLOWED_ORIGINS in production and
	// fall back to localhost dev origins otherwise. AllowCredentials is set
	// because the storefront/admin call us with a JWT cookie/bearer.
	router.Use(sharedmiddleware.HardenedCORS(sharedmiddleware.CORSConfig{
		AllowCredentials: true,
	}))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "notification-service",
			"time":    time.Now().UTC(),
		})
	})

	// Prometheus metrics endpoint — registered before Auth so scrapes are not blocked.
	router.GET("/metrics", sharedmiddleware.MetricsAuth(), gin.WrapH(metrics.Handler()))

	// JWT Auth middleware
	jwtSecret := sharedconfig.MustGetJWTSecret()
	router.Use(sharedmiddleware.Auth(sharedmiddleware.AuthConfig{SecretKey: jwtSecret}))

	// Register API routes
	api.RegisterRoutes(router, handler)
	api.RegisterTemplateRoutes(router, tmplHandler)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Infof("Notification Service listening on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
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

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
