package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ecommerce/user-service/internal/api"
	"github.com/ecommerce/user-service/internal/middleware"
	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/repository"
	"github.com/ecommerce/user-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// Initialize logger
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)

	// Load configuration from environment
	config := loadConfig()

	// Connect to database
	db, err := connectDB(config)
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to database")
	}

	// Run migrations
	if err := runMigrations(db); err != nil {
		log.WithError(err).Fatal("Failed to run migrations")
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)

	// Initialize services
	tokenConfig := models.TokenConfig{
		SecretKey:      config.JWTSecret,
		ExpirationTime: 24 * time.Hour,
		Issuer:         "user-service",
	}
	authService := service.NewAuthService(userRepo, tokenConfig, log)
	userService := service.NewUserService(userRepo, log)

	// Initialize handlers
	authHandler := api.NewAuthHandler(authService, log)
	userHandler := api.NewUserHandler(userService, log)

	// Setup router
	router := setupRouter(authHandler, userHandler, authService, log)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", config.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.WithField("port", config.Port).Info("Starting User Service")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown with 30-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.WithError(err).Fatal("Server forced to shutdown")
	}

	log.Info("Server exited")
}

// Config holds application configuration
type Config struct {
	Port       string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	return Config{
		Port:       getEnv("PORT", "8082"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "user_db"),
		JWTSecret:  getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
	}
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// connectDB connects to PostgreSQL database
func connectDB(config Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost,
		config.DBPort,
		config.DBUser,
		config.DBPassword,
		config.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})

	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// runMigrations runs database migrations
func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
	)
}

// setupRouter sets up the Gin router with all routes and middleware
func setupRouter(
	authHandler *api.AuthHandler,
	userHandler *api.UserHandler,
	authService service.AuthService,
	logger *logrus.Logger,
) *gin.Engine {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())
	router.Use(requestLoggerMiddleware(logger))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "user-service",
			"time":    time.Now().UTC(),
		})
	})

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Public authentication routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Protected authentication routes
		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.AuthMiddleware(authService))
		{
			authProtected.GET("/profile", authHandler.GetProfile)
			authProtected.POST("/change-password", authHandler.ChangePassword)
		}

		// Protected user routes
		users := v1.Group("/users")
		users.Use(middleware.AuthMiddleware(authService))
		{
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)

			// Admin-only routes
			users.PATCH("/:id/role", middleware.RequireRole(models.UserRoleAdmin), userHandler.UpdateUserRole)
			users.PATCH("/:id/status", middleware.RequireRole(models.UserRoleAdmin), userHandler.UpdateUserStatus)
		}
	}

	return router
}

// corsMiddleware adds CORS headers
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// requestLoggerMiddleware logs HTTP requests
func requestLoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		logger.WithFields(logrus.Fields{
			"method":   c.Request.Method,
			"path":     path,
			"status":   status,
			"duration": duration.Milliseconds(),
			"ip":       c.ClientIP(),
		}).Info("HTTP request")
	}
}
