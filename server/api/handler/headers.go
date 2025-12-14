package handler

import (
	"net/http"
	"os"
)

var corsAllowedOrigin string

func SetCorsHeaders(w http.ResponseWriter) {
	if corsAllowedOrigin == "" {
		corsAllowedOrigin = os.Getenv("CORS_ALLOWED_ORIGIN")
	}
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Origin", corsAllowedOrigin)
	w.Header().Set("Access-Control-Allow-Headers", "Authorization")
}

func SetNoCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
}

// CorsMiddleware adds CORS headers to all responses and handles preflight requests
func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetCorsHeaders(w)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			return
		}

		next.ServeHTTP(w, r)
	})
}
