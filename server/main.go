package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"webapp-server/api"
	"webapp-server/db"

	"github.com/joho/godotenv"
	"webapp-server/router"
)

// corsMiddleware adds CORS headers to the response
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// If this is a preflight request, the method will be OPTIONS,
		// so no further processing is needed
		if r.Method == "OPTIONS" {
			return
		}

		// Call the next handler, which can be another middleware or the final handler
		next.ServeHTTP(w, r)
	})
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading the .env file: %v", err)
	}

	db.Init()

	rtr := router.NewRouter([]router.Endpoint{
		api.External,
	})

	// Apply the CORS middleware to the router
	http.Handle("/", corsMiddleware(rtr))

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "3001"
	}

	log.Printf("Server listening on http://localhost:%s\n", port)
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), nil); err != nil {
		log.Fatalf("There was an error with the http server: %v", err)
	}
}
