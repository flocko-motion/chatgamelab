package router

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
