package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// NewLogger creates a new logger instance
func NewLogger(env string) *logrus.Logger {
	log := logrus.New()

	// Set output
	log.SetOutput(os.Stdout)

	// Set log format
	if env == "production" {
		log.SetFormatter(&logrus.JSONFormatter{})
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
		log.SetLevel(logrus.DebugLevel)
	}

	return log
}
