package httpx

import (
	"bytes"
	"cgl/log"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/getsentry/sentry-go"
)

const maxBodyCapture = 8 * 1024 // 8KB cap for captured bodies

// Middleware is a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares in order (first middleware is outermost)
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// CORS returns a middleware that handles CORS preflight and sets CORS headers
// Uses the request Origin header to support credentials (cookies)
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		SetCORSHeadersWithOrigin(w, origin)

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Logging returns a middleware that logs requests
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Buffer request body so we can include it in error reports
		var reqBody []byte
		if r.Body != nil {
			reqBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(reqBody))
		}

		// Wrap ResponseWriter to capture status code and response body
		wrapped := &capturingResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(wrapped, r)

		// Skip logging CORS preflight requests
		if r.Method == http.MethodOptions {
			return
		}

		duration := time.Since(start)
		log.Info(fmt.Sprintf("%s %s → %d (%s)", r.Method, r.URL.Path, wrapped.status, formatDuration(duration)))

		// Report error responses (4xx/5xx) to Sentry
		if wrapped.status >= 400 {
			sentryLevel := sentry.LevelWarning
			if wrapped.status >= 500 {
				sentryLevel = sentry.LevelError
			}
			sentry.WithScope(func(scope *sentry.Scope) {
				scope.SetTag("http.method", r.Method)
				scope.SetTag("http.path", r.URL.Path)
				scope.SetExtra("http.status", wrapped.status)
				scope.SetExtra("http.duration_ms", duration.Milliseconds())

				// Request details
				scope.SetExtra("request.headers", flattenHeaders(r.Header))
				if r.URL.RawQuery != "" {
					scope.SetExtra("request.query", r.URL.RawQuery)
				}
				if len(reqBody) > 0 {
					scope.SetExtra("request.body", truncateBody(reqBody))
				}

				// Response body (text only, capped)
				if respBody := wrapped.capturedBody(); respBody != "" {
					scope.SetExtra("response.body", respBody)
				}

				scope.SetLevel(sentryLevel)
				sentry.CaptureMessage(fmt.Sprintf("%s %s → %d", r.Method, r.URL.Path, wrapped.status))
			})
		}
	})
}

// capturingResponseWriter wraps http.ResponseWriter to capture status code and response body.
type capturingResponseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *capturingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *capturingResponseWriter) Write(b []byte) (int, error) {
	// Only buffer if we haven't exceeded the cap yet
	if w.body.Len() < maxBodyCapture {
		remaining := maxBodyCapture - w.body.Len()
		if len(b) <= remaining {
			w.body.Write(b)
		} else {
			w.body.Write(b[:remaining])
		}
	}
	return w.ResponseWriter.Write(b)
}

// Flush implements http.Flusher to support SSE streaming
func (w *capturingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// capturedBody returns the captured response body as a string.
// Returns empty string for binary content.
func (w *capturingResponseWriter) capturedBody() string {
	if w.body.Len() == 0 {
		return ""
	}
	b := w.body.Bytes()
	if !utf8.Valid(b) {
		return ""
	}
	s := string(b)
	if w.body.Len() >= maxBodyCapture {
		s += "\n... [truncated]"
	}
	return s
}

// truncateBody returns the request body as a string, capped at maxBodyCapture.
// Returns "[binary]" for non-text content.
func truncateBody(b []byte) string {
	if !utf8.Valid(b) {
		return "[binary]"
	}
	if len(b) > maxBodyCapture {
		return string(b[:maxBodyCapture]) + "\n... [truncated]"
	}
	return string(b)
}

// formatDuration formats a duration in a compact human-readable form.
func formatDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// flattenHeaders converts http.Header to a simple map for Sentry extras.
// Redacts the Authorization header.
func flattenHeaders(h http.Header) map[string]string {
	result := make(map[string]string, len(h))
	for k, v := range h {
		lower := strings.ToLower(k)
		if lower == "authorization" || lower == "cookie" {
			result[k] = "[redacted]"
		} else {
			result[k] = strings.Join(v, ", ")
		}
	}
	return result
}

// Recover returns a middleware that recovers from panics
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered",
					"method", r.Method,
					"path", r.URL.Path,
					"error", err,
				)

				sentry.WithScope(func(scope *sentry.Scope) {
					scope.SetTag("http.method", r.Method)
					scope.SetTag("http.path", r.URL.Path)
					scope.SetLevel(sentry.LevelFatal)
					if e, ok := err.(error); ok {
						sentry.CaptureException(e)
					} else {
						sentry.CaptureMessage(fmt.Sprintf("panic: %v", err))
					}
				})

				WriteError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// NoCache returns a middleware that sets no-cache headers
func NoCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetNoCacheHeaders(w)
		next.ServeHTTP(w, r)
	})
}
