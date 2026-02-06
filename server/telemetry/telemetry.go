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

// CaptureError sends an error event to Sentry with optional extra context.
func CaptureError(err error, tags map[string]string, extras map[string]interface{}) {
	sentry.WithScope(func(scope *sentry.Scope) {
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		for k, v := range extras {
			scope.SetExtra(k, v)
		}
		sentry.CaptureException(err)
	})
}

// CaptureMessage sends a message event to Sentry.
func CaptureMessage(msg string, level sentry.Level) {
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)
		sentry.CaptureMessage(msg)
	})
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
