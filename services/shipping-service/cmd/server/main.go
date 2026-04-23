package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ecommerce/shipping-service/internal/api"
	"github.com/ecommerce/shipping-service/internal/config"
	"github.com/ecommerce/shipping-service/internal/models"
	"github.com/ecommerce/shipping-service/internal/repository"
	"github.com/ecommerce/shipping-service/internal/service"
	"github.com/ecommerce/shipping-service/pkg/database"
	"github.com/ecommerce/shipping-service/pkg/logger"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	log := logger.NewLogger(cfg.Server.Env)
	log.Info("Starting Shipping Service...")

	// Connect to database
	db, err := database.NewPostgresDB(cfg, log)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := database.RunMigrations(db, &models.Shipment{}, &models.ShipmentItem{}, &models.ShipmentEvent{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Info("Database migrations completed")

	// Initialize repository
	shipmentRepo := repository.NewShipmentRepository(db)

	// Initialize carrier service (simulated for development)
	carrierService := service.NewSimulatedCarrierService()

	// Initialize shipping service
	shippingService := service.NewShippingService(shipmentRepo, carrierService, log)

	// Initialize handler
	handler := api.NewShippingHandler(shippingService, log)

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

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "shipping-service",
			"time":    time.Now().UTC(),
		})
	})

	// Register API routes
	api.RegisterRoutes(router, handler)

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
		log.Infof("Shipping Service listening on port %s", cfg.Server.Port)
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
