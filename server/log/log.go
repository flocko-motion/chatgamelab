package log

import (
	"log/slog"
	"os"
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
	os.Exit(1)
}
