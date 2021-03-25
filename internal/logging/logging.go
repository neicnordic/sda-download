package logging

import (
	"os"

	log "github.com/sirupsen/logrus"
)

// determineLogLevel converts string representation of log level to log.Level
func determineLogLevel(level string) log.Level {
	switch level {
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	case "info":
		return log.InfoLevel
	case "panic":
		return log.PanicLevel
	case "warn":
		return log.WarnLevel
	case "trace":
		return log.TraceLevel
	case "debug":
		return log.DebugLevel
	default:
		return log.DebugLevel
	}
}

// LoggingSetup configures logging format and rules
func LoggingSetup(logLevel string) {
	// Log formatting
	log.SetFormatter(&log.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)
	log.Info(logLevel)
	// Minimum message level
	log.SetLevel(determineLogLevel(logLevel))
}
