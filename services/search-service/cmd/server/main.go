package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ecommerce/search-service/internal/api"
	"github.com/ecommerce/search-service/internal/config"
	"github.com/ecommerce/search-service/internal/messaging"
	"github.com/ecommerce/search-service/internal/repository"
	"github.com/ecommerce/search-service/internal/service"
	"github.com/ecommerce/search-service/pkg/logger"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	log := logger.NewLogger(cfg.Server.Env)
	log.Info("Starting Search Service...")

	// Connect to Elasticsearch
	esCfg := elasticsearch.Config{
		Addresses: []string{cfg.Elasticsearch.URL},
	}

	esClient, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		log.Fatalf("Failed to create Elasticsearch client: %v", err)
	}

	// Verify connection
	res, err := esClient.Info()
	if err != nil {
		log.Fatalf("Failed to connect to Elasticsearch: %v", err)
	}
	defer res.Body.Close()
	log.Info("Successfully connected to Elasticsearch")

	// Initialize repository
	searchRepo := repository.NewSearchRepository(esClient, cfg.Elasticsearch.Index, log)

	// Ensure index exists
	if err := searchRepo.EnsureIndex(context.Background()); err != nil {
		log.Fatalf("Failed to ensure Elasticsearch index: %v", err)
	}

	// Initialize service
	searchService := service.NewSearchService(searchRepo, log)

	// Initialize Kafka consumer
	consumer := messaging.NewEventConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, searchService, log)
	consumer.Start(context.Background())
	defer consumer.Stop()

	// Initialize handler
	handler := api.NewSearchHandler(searchService, log)

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
			"service": "search-service",
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
		log.Infof("Search Service listening on port %s", cfg.Server.Port)
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
