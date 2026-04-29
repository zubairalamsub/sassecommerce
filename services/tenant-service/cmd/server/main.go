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
	"github.com/ecommerce/tenant-service/internal/middleware"
	"github.com/ecommerce/tenant-service/internal/models"
	"github.com/ecommerce/tenant-service/internal/repository"
	"github.com/ecommerce/tenant-service/internal/service"
	"github.com/ecommerce/tenant-service/pkg/database"
	"github.com/ecommerce/tenant-service/pkg/kafka"
	"github.com/ecommerce/tenant-service/pkg/logger"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	defer kafkaProducer.Close()

	// Initialize repositories
	tenantRepo := repository.NewTenantRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	// Initialize services
	auditService := service.NewAuditService(auditRepo, log)
	tenantService := service.NewTenantService(tenantRepo, kafkaProducer, log)

	// Initialize handlers
	tenantHandler := api.NewTenantHandler(tenantService, log)
	auditHandler := api.NewAuditHandler(auditService, log)

	// Setup Gin router
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Tenant-Id"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
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

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		tenants := v1.Group("/tenants")
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

		// Audit logs
		v1.GET("/audit-logs", auditHandler.ListAuditLogs)
		v1.GET("/audit-logs/:id", auditHandler.GetAuditLog)
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
