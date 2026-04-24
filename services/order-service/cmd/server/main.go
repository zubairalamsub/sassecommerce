package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/yourusername/ecommerce/order-service/internal/api"
	"github.com/yourusername/ecommerce/order-service/internal/config"
	"github.com/yourusername/ecommerce/order-service/internal/domain/commands"
	"github.com/yourusername/ecommerce/order-service/internal/eventstore"
	"github.com/yourusername/ecommerce/order-service/internal/messaging"
	"github.com/yourusername/ecommerce/order-service/internal/projection"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	logger, err := initLogger(cfg.LogLevel)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting Order Service",
		zap.String("version", "1.0.0"),
		zap.String("log_level", cfg.LogLevel),
	)

	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	logger.Info("Database connection established")

	// Initialize event store
	eventStore, err := eventstore.NewPostgresEventStore(db)
	if err != nil {
		logger.Fatal("Failed to initialize event store", zap.Error(err))
	}

	logger.Info("Event store initialized")

	// Initialize projection
	orderProjection, err := projection.NewOrderProjection(db, logger)
	if err != nil {
		logger.Fatal("Failed to initialize projection", zap.Error(err))
	}

	logger.Info("Order projection initialized")

	// Initialize Kafka components if enabled
	var finalEventStore eventstore.EventStore = eventStore
	var consumer messaging.EventConsumer
	var externalConsumer *messaging.ExternalEventConsumer

	if cfg.Kafka.Enabled {
		// Initialize Kafka publisher
		publisher := messaging.NewKafkaEventPublisher(
			cfg.Kafka.Brokers,
			cfg.Kafka.Topic,
			logger,
		)
		defer publisher.Close()

		logger.Info("Kafka publisher initialized",
			zap.Strings("brokers", cfg.Kafka.Brokers),
			zap.String("topic", cfg.Kafka.Topic),
		)

		// Wrap event store with Kafka publishing
		finalEventStore = eventstore.NewEventStoreWithKafka(
			eventStore,
			publisher,
			logger,
		)

		// Initialize Kafka consumer (internal order-events for CQRS projections)
		consumer = messaging.NewKafkaEventConsumer(
			cfg.Kafka.Brokers,
			cfg.Kafka.Topic,
			cfg.Kafka.ConsumerGroup,
			orderProjection,
			logger,
		)

		// Start consumer in background
		ctx := context.Background()
		if err := consumer.Start(ctx); err != nil {
			logger.Fatal("Failed to start Kafka consumer", zap.Error(err))
		}
		defer consumer.Stop()

		logger.Info("Kafka consumer started",
			zap.String("group_id", cfg.Kafka.ConsumerGroup),
		)

	} else {
		logger.Warn("Kafka is disabled - events will not be published")
	}

	// Initialize command handler
	commandHandler := commands.NewCommandHandler(finalEventStore, logger)

	// Start external event consumer (payment-events, inventory-events, shipping-events)
	if cfg.Kafka.Enabled {
		adapter := &commandAdapter{handler: commandHandler}
		externalConsumer = messaging.NewExternalEventConsumer(
			cfg.Kafka.Brokers,
			"order-service-external",
			adapter,
			logger,
		)
		ctx := context.Background()
		if err := externalConsumer.Start(ctx); err != nil {
			logger.Fatal("Failed to start external event consumer", zap.Error(err))
		}
		logger.Info("External event consumer started")
	}

	// Initialize API handlers
	apiCommandHandler := api.NewCommandHandler(
		commandHandler,
		finalEventStore,
		logger,
		cfg.Services.InventoryURL,
		cfg.Services.PaymentURL,
	)

	queryHandler := api.NewQueryHandler(orderProjection, logger)

	// Setup router
	router := api.NewRouter(apiCommandHandler, queryHandler, logger)
	engine := router.Setup()

	// Start HTTP server
	serverAddr := cfg.GetServerAddress()
	logger.Info("Starting HTTP server", zap.String("address", serverAddr))

	// Graceful shutdown
	go func() {
		if err := engine.Run(serverAddr); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Cleanup
	if consumer != nil {
		consumer.Stop()
	}
	if externalConsumer != nil {
		externalConsumer.Stop()
	}

	logger.Info("Server shutdown complete")
}

// initLogger initializes the logger
func initLogger(level string) (*zap.Logger, error) {
	var zapLevel zap.AtomicLevel

	switch level {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	config := zap.Config{
		Level:            zapLevel,
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	return config.Build()
}

// commandAdapter adapts messaging.OrderCommand to commands.Command for the CommandHandler.
// This breaks the import cycle between messaging and commands packages.
type commandAdapter struct {
	handler *commands.CommandHandler
}

func (a *commandAdapter) Handle(cmd messaging.OrderCommand) error {
	switch c := cmd.(type) {
	case messaging.ConfirmOrderCmd:
		return a.handler.Handle(commands.ConfirmOrderCommand{
			OrderID:     c.OrderID,
			ConfirmedBy: c.ConfirmedBy,
		})
	case messaging.CancelOrderCmd:
		return a.handler.Handle(commands.CancelOrderCommand{
			OrderID:     c.OrderID,
			Reason:      c.Reason,
			CancelledBy: c.CancelledBy,
		})
	case messaging.ShipOrderCmd:
		return a.handler.Handle(commands.ShipOrderCommand{
			OrderID:        c.OrderID,
			TrackingNumber: c.TrackingNumber,
			Carrier:        c.Carrier,
			ShippedBy:      c.ShippedBy,
		})
	case messaging.DeliverOrderCmd:
		return a.handler.Handle(commands.DeliverOrderCommand{
			OrderID:    c.OrderID,
			ReceivedBy: c.ReceivedBy,
		})
	default:
		return fmt.Errorf("unknown external command type: %T", cmd)
	}
}

// initDatabase initializes the database connection
func initDatabase(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.GetDatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
