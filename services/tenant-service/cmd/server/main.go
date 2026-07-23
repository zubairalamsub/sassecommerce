package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ecommerce/tenant-service/internal/api"
	"github.com/ecommerce/tenant-service/internal/config"
	"github.com/ecommerce/tenant-service/internal/messaging"
	"github.com/ecommerce/tenant-service/internal/middleware"
	"github.com/ecommerce/tenant-service/internal/models"
	"github.com/ecommerce/tenant-service/internal/repository"
	"github.com/ecommerce/tenant-service/internal/service"
	"github.com/ecommerce/tenant-service/pkg/database"
	"github.com/ecommerce/tenant-service/pkg/kafka"
	"github.com/ecommerce/tenant-service/pkg/logger"

	sharedconfig "github.com/ecommerce/shared/go/pkg/config"
	"github.com/ecommerce/shared/go/pkg/metrics"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.NewLogger(cfg.Server.Env)
	log.Info("Starting Tenant Service...")

	// Connect to database
	db, err := database.NewPostgresDB(cfg, log)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.RunMigrations(db, &models.Tenant{}, &models.AuditLog{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Info("Database migrations completed")

	// Initialize Kafka producer
	kafkaProducer := kafka.NewProducer(cfg.Kafka.Brokers, log)
	defer func() { _ = kafkaProducer.Close() }()

	// Optional Redis cache for tenant lookups (id/slug/domain). If Redis is
	// unreachable the service runs without caching (repo gets a nil client).
	var redisClient *redis.Client
	if cfg.Redis.Host != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
			Password: cfg.Redis.Password,
		})
		pingCtx, cancelPing := context.WithTimeout(context.Background(), 5*time.Second)
		if err := redisClient.Ping(pingCtx).Err(); err != nil {
			log.WithError(err).Warn("Redis unavailable; continuing without tenant cache")
			redisClient = nil
		} else {
			log.Info("Tenant cache enabled (Redis)")
		}
		cancelPing()
		defer func() {
			if redisClient != nil {
				_ = redisClient.Close()
			}
		}()
	}

	// Initialize repositories
	tenantRepo := repository.NewTenantRepository(db, redisClient)
	auditRepo := repository.NewAuditRepository(db)
	usageRepo := repository.NewUsageRepository(db)

	// Initialize services
	auditService := service.NewAuditService(auditRepo, log)
	tenantService := service.NewTenantService(tenantRepo, kafkaProducer, log)
	usageService := service.NewUsageService(usageRepo, log)

	// Initialize handlers
	tenantHandler := api.NewTenantHandler(tenantService, log)
	auditHandler := api.NewAuditHandler(auditService, log)
	usageHandler := api.NewUsageHandler(usageService, log)

	// Start centralised audit event consumer (listens to all service Kafka topics)
	auditConsumer := messaging.NewAuditEventConsumer(cfg.Kafka.Brokers, auditService, log)
	auditConsumer.Start(context.Background())
	defer auditConsumer.Close()

	// Setup Gin router
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	// Security headers land early so they apply to every response — including
	// errors raised by middleware further down the chain.
	router.Use(sharedmiddleware.SecurityHeaders(sharedmiddleware.SecurityHeadersConfig{}))
	router.Use(metrics.Middleware("tenant-service"))
	router.Use(sharedmiddleware.RequestLogger(sharedmiddleware.RequestLoggerConfig{
		Logger:          log,
		LogRequestBody:  true,
		LogResponseBody: true,
		SkipPaths:       []string{"/health", "/ready", "/metrics"},
	}))

	// Configure CORS
	// CORS: origins come from CORS_ALLOWED_ORIGINS (comma-separated) in
	// production and fall back to localhost dev origins otherwise. Wildcards
	// are rejected in production and credentials only sent to explicit origins.
	router.Use(sharedmiddleware.HardenedCORS(sharedmiddleware.CORSConfig{
		AllowCredentials: true,
	}))

	// Add audit middleware
	router.Use(middleware.AuditMiddleware(auditService, log))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "tenant-service",
			"time":    time.Now().UTC(),
		})
	})

	// Prometheus metrics endpoint — registered before Auth so scrapes are not blocked.
	router.GET("/metrics", sharedmiddleware.MetricsAuth(), gin.WrapH(metrics.Handler()))

	// All /api/v1 routes require a valid JWT. This is the platform control
	// plane, so it must never be reachable anonymously.
	jwtSecret := sharedconfig.MustGetJWTSecret()
	v1 := router.Group("/api/v1")
	v1.Use(sharedmiddleware.Auth(sharedmiddleware.AuthConfig{SecretKey: jwtSecret}))
	{
		// Tenant lifecycle is a platform-operator (super_admin) concern:
		// managing the set of tenants is inherently cross-tenant.
		tenants := v1.Group("/tenants")
		tenants.Use(sharedmiddleware.RequireRole("super_admin"))
		{
			tenants.POST("", tenantHandler.CreateTenant)
			tenants.GET("", tenantHandler.ListTenants)
			tenants.GET("/:id", tenantHandler.GetTenant)
			tenants.GET("/slug/:slug", tenantHandler.GetTenantBySlug)
			tenants.GET("/domain", tenantHandler.GetTenantByDomain)
			tenants.PUT("/:id", tenantHandler.UpdateTenant)
			tenants.PATCH("/:id/config", tenantHandler.UpdateTenantConfig)
			tenants.DELETE("/:id", tenantHandler.DeleteTenant)
		}

		// Audit logs: a tenant admin sees only their own tenant's logs
		// (handler derives the tenant from the JWT); a super_admin may scope
		// to any tenant. Requires at least admin.
		auditLogs := v1.Group("/audit-logs")
		auditLogs.Use(sharedmiddleware.RequireRole("admin", "super_admin"))
		{
			auditLogs.GET("", auditHandler.ListAuditLogs)
			auditLogs.GET("/:id", auditHandler.GetAuditLog)
		}

		// Cross-tenant usage report — super_admin only.
		v1.GET("/admin/usage", sharedmiddleware.RequireRole("super_admin"), usageHandler.GetUsage)
	}

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
		log.Infof("Tenant Service listening on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Server exited")
}
