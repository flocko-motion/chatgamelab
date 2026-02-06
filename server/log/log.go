package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

var (
	logger *slog.Logger
	level  = new(slog.LevelVar)
)

func init() {
	// Default to Info level, can be changed with SetDebug
	level.Set(slog.LevelInfo)

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger = slog.New(handler)
	slog.SetDefault(logger)
}

// SetDebug enables or disables debug logging
func SetDebug(enabled bool) {
	if enabled {
		level.Set(slog.LevelDebug)
	} else {
		level.Set(slog.LevelInfo)
	}
}

// SetLevel sets the log level from a string (debug, info, warn, error)
func SetLevel(levelStr string) {
	switch levelStr {
	case "debug":
		level.Set(slog.LevelDebug)
	case "info":
		level.Set(slog.LevelInfo)
	case "warn":
		level.Set(slog.LevelWarn)
	case "error":
		level.Set(slog.LevelError)
	default:
		level.Set(slog.LevelInfo)
	}
}

// Logger returns the configured slog logger
func Logger() *slog.Logger {
	return logger
}

// With returns a logger with additional attributes
func With(args ...any) *slog.Logger {
	return logger.With(args...)
}

// Debug logs at debug level
func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// Info logs at info level
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// Warn logs at warn level
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Error logs at error level
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

// Fatal logs at error level and exits with status code 1
func Fatal(msg string, args ...any) {
	logger.Error(msg, args...)
	sentry.Flush(2 * time.Second)
	os.Exit(1)
}

// EnableSentry wraps the current slog handler so that Warn/Error messages
// are also forwarded to Sentry. Call this after sentry.Init() succeeds.
func EnableSentry() {
	logger = slog.New(&sentryHandler{inner: logger.Handler()})
	slog.SetDefault(logger)
}

// sentryHandler wraps an slog.Handler and forwards Warn+ records to Sentry.
type sentryHandler struct {
	inner slog.Handler
}

func (h *sentryHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}

func (h *sentryHandler) Handle(ctx context.Context, r slog.Record) error {
	// Always delegate to the inner handler first
	err := h.inner.Handle(ctx, r)

	// Forward warnings and errors to Sentry
	if r.Level >= slog.LevelWarn {
		sentryLevel := sentry.LevelWarning
		if r.Level >= slog.LevelError {
			sentryLevel = sentry.LevelError
		}

		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetLevel(sentryLevel)
			r.Attrs(func(a slog.Attr) bool {
				scope.SetExtra(a.Key, a.Value.String())
				return true
			})
			sentry.CaptureMessage(fmt.Sprintf("%s: %s", r.Level, r.Message))
		})
	}

	return err
}

func (h *sentryHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &sentryHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *sentryHandler) WithGroup(name string) slog.Handler {
	return &sentryHandler{inner: h.inner.WithGroup(name)}
}
