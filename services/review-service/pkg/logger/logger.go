package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

func NewLogger(env string) *logrus.Logger {
	log := logrus.New()
	log.SetOutput(os.Stdout)

	if env == "production" {
		log.SetFormatter(&logrus.JSONFormatter{})
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
		log.SetLevel(logrus.DebugLevel)
	}

	return log
}
