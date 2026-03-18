package telemetry

import (
	"cgl/log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

// Init initializes Sentry/GlitchTip error reporting.
// Does nothing if SENTRY_DSN_BACKEND is not set.
func Init(version string) {
	dsn := os.Getenv("SENTRY_DSN_BACKEND")
	if dsn == "" {
		log.Info("telemetry disabled (SENTRY_DSN_BACKEND not set)")
		return
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Release:          version,
		Environment:      getEnvironment(),
		SampleRate:       1.0,
		TracesSampleRate: 0.1,
	})
	if err != nil {
		log.Error("failed to initialize sentry", "error", err)
		return
	}

	log.EnableSentry()
	log.Info("telemetry enabled", "environment", getEnvironment())
}

// Flush waits for buffered events to be sent. Call before shutdown.
func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env != "" {
		return env
	}
	devMode := os.Getenv("DEV_MODE")
	if devMode == "true" || devMode == "1" {
		return "development"
	}
	return "production"
}
