package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ecommerce/config-service/internal/api"
	"github.com/ecommerce/config-service/internal/config"
	"github.com/ecommerce/config-service/internal/models"
	"github.com/ecommerce/config-service/internal/repository"
	"github.com/ecommerce/config-service/internal/seed"
	"github.com/ecommerce/config-service/internal/service"
	"github.com/ecommerce/config-service/pkg/logger"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()
	log := logger.NewLogger(cfg.Server.Env)
	log.Info("Starting Configuration Service...")

	// Connect to PostgreSQL
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Info("Successfully connected to PostgreSQL")

	// Auto-migrate
	if err := db.AutoMigrate(
		&models.ConfigEntry{},
		&models.ConfigAuditLog{},
		&models.Menu{},
		&models.MenuItem{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Initialize repositories
	configRepo := repository.NewConfigRepository(db)
	menuRepo := repository.NewMenuRepository(db)

	// Initialize services
	configService := service.NewConfigService(configRepo, log)
	menuService := service.NewMenuService(menuRepo, log)

	// Seed default configurations
	seed.SeedDefaults(context.Background(), configService, log)

	// Seed default menus for demo tenant
	seed.SeedDefaultMenus(context.Background(), menuRepo, "default", log)

	// Initialize handlers
	handler := api.NewConfigHandler(configService, log)
	menuHandler := api.NewMenuHandler(menuService, log)

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
			"service": "config-service",
			"time":    time.Now().UTC(),
		})
	})

	api.RegisterRoutes(router, handler)
	api.RegisterMenuRoutes(router, menuHandler)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Infof("Configuration Service listening on port %s", cfg.Server.Port)
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
