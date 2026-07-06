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

	"github.com/ecommerce/user-service/internal/api"
	"github.com/ecommerce/user-service/internal/messaging"
	"github.com/ecommerce/user-service/internal/middleware"
	"github.com/ecommerce/user-service/internal/models"
	"github.com/ecommerce/user-service/internal/repository"
	"github.com/ecommerce/user-service/internal/service"
	sharedconfig "github.com/ecommerce/shared/go/pkg/config"
	"github.com/ecommerce/shared/go/pkg/metrics"
	sharedmiddleware "github.com/ecommerce/shared/go/pkg/middleware"
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

	// Initialize Kafka producer
	kafkaBrokers := strings.Split(getEnv("KAFKA_BROKERS", "kafka:9092"), ",")
	kafkaProducer := messaging.NewProducer(kafkaBrokers, log)
	defer kafkaProducer.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	wishlistRepo := repository.NewWishlistRepository(db)
	loginAttemptRepo := repository.NewLoginAttemptRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)

	// 2FA TOTP secrets are encrypted at rest with TWO_FACTOR_ENCRYPTION_KEY
	// (16/24/32 raw bytes). Running without it stores secrets base64-only,
	// acceptable for local development but never in production.
	twoFactorKey := []byte(os.Getenv("TWO_FACTOR_ENCRYPTION_KEY"))
	if len(twoFactorKey) == 0 {
		if sharedconfig.IsProduction() {
			log.Fatal("TWO_FACTOR_ENCRYPTION_KEY is required in production")
		}
		log.Warn("TWO_FACTOR_ENCRYPTION_KEY is not set — 2FA secrets will be stored unencrypted (dev only)")
	}
	twoFactorRepo, err := repository.NewTwoFactorRepository(db, twoFactorKey)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize 2FA repository")
	}

	// Initialize services
	tokenConfig := models.TokenConfig{
		SecretKey:      config.JWTSecret,
		ExpirationTime: 24 * time.Hour,
		Issuer:         "user-service",
	}
	twoFactorService := service.NewTwoFactorService(twoFactorRepo, log,
		service.WithTwoFactorKafkaPublisher(kafkaProducer))
	authService := service.NewAuthServiceWithOptions(
		userRepo,
		tokenConfig,
		kafkaProducer,
		log,
		service.WithTokenRepository(tokenRepo),
		service.WithLoginAttemptRepository(loginAttemptRepo),
		service.WithLockoutConfig(service.LockoutConfigFromEnv()),
		service.WithRefreshTokenRepository(refreshTokenRepo),
		service.WithTwoFactorService(twoFactorService),
	)
	userService := service.NewUserService(userRepo, kafkaProducer, log)

	// Initialize handlers
	authHandler := api.NewAuthHandler(authService, log)
	twoFactorHandler := api.NewTwoFactorHandler(twoFactorService, authService, log)
	userHandler := api.NewUserHandler(userService, log)
	wishlistHandler := api.NewWishlistHandler(wishlistRepo, log)

	// Setup router
	router := setupRouter(authHandler, twoFactorHandler, userHandler, wishlistHandler, authService, log)

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
		JWTSecret:  sharedconfig.MustGetJWTSecret(),
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
		&models.VerificationToken{},
		&models.PasswordResetToken{},
		&models.WishlistItem{},
		&models.LoginAttempt{},
	)
}

// setupRouter sets up the Gin router with all routes and middleware
func setupRouter(
	authHandler *api.AuthHandler,
	twoFactorHandler *api.TwoFactorHandler,
	userHandler *api.UserHandler,
	wishlistHandler *api.WishlistHandler,
	authService service.AuthService,
	logger *logrus.Logger,
) *gin.Engine {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Global middleware
	router.Use(gin.Recovery())
	// Security headers land early so they apply to every response — including
	// errors raised by middleware further down the chain.
	router.Use(sharedmiddleware.SecurityHeaders(sharedmiddleware.SecurityHeadersConfig{}))
	router.Use(metrics.Middleware("user-service"))
	router.Use(sharedmiddleware.HardenedCORS(sharedmiddleware.CORSConfig{
		AllowCredentials: true,
	}))
	router.Use(sharedmiddleware.RequestLogger(sharedmiddleware.RequestLoggerConfig{
		Logger:          logger,
		LogRequestBody:  true,
		LogResponseBody: true,
		SkipPaths:       []string{"/health", "/ready", "/metrics"},
	}))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "user-service",
			"time":    time.Now().UTC(),
		})
	})

	// Prometheus metrics endpoint — registered before Auth/RateLimit so scrapes are not blocked.
	router.GET("/metrics", sharedmiddleware.MetricsAuth(), gin.WrapH(metrics.Handler()))

	// Rate limiting (after /metrics so Prometheus scrapes aren't throttled)
	router.Use(sharedmiddleware.RateLimit(sharedmiddleware.RateLimitConfig{
		Rate:   100,
		Window: time.Minute,
	}))

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Public authentication routes — stricter per-IP rate limit because
		// these endpoints (login/register/forgot-password) are the prime
		// targets for credential-stuffing and email-enumeration scans.
		auth := v1.Group("/auth")
		auth.Use(sharedmiddleware.RateLimit(sharedmiddleware.RateLimitConfig{
			Rate:   20,
			Window: time.Minute,
		}))
		{
			// Per-route limits on top of the group limit. IP-keyed limits
			// slow single-source brute force; body-field-keyed limits (email,
			// token) throttle attacks distributed across many IPs against a
			// single account, and cap reset/verification email spam per target.
			auth.POST("/register",
				sharedmiddleware.RateLimit(sharedmiddleware.RateLimitConfig{Rate: 3, Window: time.Hour}),
				authHandler.Register)
			auth.POST("/login",
				sharedmiddleware.RateLimit(sharedmiddleware.RateLimitConfig{Rate: 5, Window: time.Minute}),
				sharedmiddleware.RateLimitByBodyField(20, time.Hour, "email"),
				authHandler.Login)
			auth.POST("/verify-email",
				sharedmiddleware.RateLimitByBodyField(5, time.Hour, "token"),
				authHandler.VerifyEmail)
			auth.POST("/forgot-password",
				sharedmiddleware.RateLimitByBodyField(3, time.Hour, "email"),
				authHandler.ForgotPassword)
			auth.POST("/reset-password",
				sharedmiddleware.RateLimitByBodyField(5, time.Hour, "token"),
				authHandler.ResetPassword)
			auth.POST("/resend-verification",
				sharedmiddleware.RateLimitByBodyField(3, time.Hour, "email"),
				authHandler.ResendVerification)
			// Complete a 2FA-gated login. Limited per challenge token so a
			// TOTP code cannot be brute-forced within the challenge TTL.
			auth.POST("/login/2fa",
				sharedmiddleware.RateLimit(sharedmiddleware.RateLimitConfig{Rate: 10, Window: time.Minute}),
				sharedmiddleware.RateLimitByBodyField(5, 5*time.Minute, "challenge_token"),
				authHandler.VerifyTwoFactor)
			// Rotate a refresh token for a new access+refresh pair.
			auth.POST("/refresh",
				sharedmiddleware.RateLimit(sharedmiddleware.RateLimitConfig{Rate: 30, Window: time.Minute}),
				authHandler.Refresh)
		}

		// Protected authentication routes
		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.AuthMiddleware(authService))
		{
			authProtected.GET("/profile", authHandler.GetProfile)
			authProtected.POST("/change-password", authHandler.ChangePassword)
			authProtected.POST("/logout", authHandler.Logout)
			authProtected.POST("/logout-all", authHandler.LogoutAll)
			authProtected.GET("/sessions", authHandler.ListSessions)
			authProtected.DELETE("/sessions/:id", authHandler.RevokeSession)

			// TOTP two-factor enrollment management
			authProtected.GET("/2fa", twoFactorHandler.Status)
			authProtected.POST("/2fa/setup", twoFactorHandler.BeginSetup)
			authProtected.POST("/2fa/confirm", twoFactorHandler.ConfirmSetup)
			authProtected.POST("/2fa/disable", twoFactorHandler.Disable)
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

		// Wishlist routes (require auth)
		wishlist := v1.Group("/wishlist")
		wishlist.Use(middleware.AuthMiddleware(authService))
		wishlist.Use(middleware.RequireTenant())
		{
			wishlist.GET("", wishlistHandler.GetWishlist)
			wishlist.POST("/items", wishlistHandler.AddWishlistItem)
			wishlist.DELETE("/items/:productId", wishlistHandler.RemoveWishlistItem)
			wishlist.DELETE("", wishlistHandler.ClearWishlist)
		}
	}

	return router
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
