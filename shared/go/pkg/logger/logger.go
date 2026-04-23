package logger

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus.Logger with additional functionality
type Logger struct {
	*logrus.Logger
}

// Config holds logger configuration
type Config struct {
	Level      string
	Format     string // "json" or "text"
	Output     string // "stdout" or file path
	ServiceName string
}

// New creates a new logger instance
func New(config Config) *Logger {
	log := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Set formatter
	if config.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	// Set output
	if config.Output != "" && config.Output != "stdout" {
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
		} else {
			log.Warn("Failed to log to file, using stdout")
		}
	}

	// Add service name as default field
	if config.ServiceName != "" {
		log = log.WithField("service", config.ServiceName).Logger
	}

	return &Logger{log}
}

// WithContext returns a logger with context fields
func (l *Logger) WithContext(fields map[string]interface{}) *logrus.Entry {
	return l.WithFields(fields)
}

// WithRequestID returns a logger with request ID
func (l *Logger) WithRequestID(requestID string) *logrus.Entry {
	return l.WithField("request_id", requestID)
}

// WithTenantID returns a logger with tenant ID
func (l *Logger) WithTenantID(tenantID string) *logrus.Entry {
	return l.WithField("tenant_id", tenantID)
}

// WithUserID returns a logger with user ID
func (l *Logger) WithUserID(userID string) *logrus.Entry {
	return l.WithField("user_id", userID)
}

// WithError returns a logger with error field
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// Default logger instance
var defaultLogger *Logger

// Init initializes the default logger
func Init(config Config) {
	defaultLogger = New(config)
}

// Get returns the default logger
func Get() *Logger {
	if defaultLogger == nil {
		defaultLogger = New(Config{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		})
	}
	return defaultLogger
}
