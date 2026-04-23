package logger

import (
	"github.com/sirupsen/logrus"
)

func NewLogger(env string) *logrus.Logger {
	log := logrus.New()

	if env == "production" {
		log.SetFormatter(&logrus.JSONFormatter{})
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
		log.SetLevel(logrus.DebugLevel)
	}

	return log
}
