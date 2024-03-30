package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
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
		log.Println("No .env file found - not importing env variables from file")
	}

	db.Init()

	theRouter := router.NewRouter([]router.Endpoint{
		api.Game,
		api.Games,
		api.Image,
		api.Session,
		api.Status,
		api.Upgrade,
		api.User,
		api.PublicGame,
		api.PublicSession,
	})

	htmlDir := http.Dir("./html")
	theRouter.Handle("/", spaHandler(htmlDir, "./html/index.html"))

	// Apply the CORS middleware to the router
	http.Handle("/", corsMiddleware(theRouter))

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server listening on http://localhost:%s\n", port)
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), nil); err != nil {
		log.Fatalf("There was an error with the http server: %v", err)
	}
}

// spaHandler is a custom http handler that serves the SPA
func spaHandler(htmlDir http.FileSystem, indexFileName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Serve the file directly if it exists and is not a directory
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, r)
			return
		}

		_, err := htmlDir.Open(path)
		if os.IsNotExist(err) {
			// File does not exist, serve index.html
			http.ServeFile(w, r, indexFileName)
		} else if err != nil {
			// Some other error occurred, send an internal server error
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			// File exists, serve it as usual
			http.FileServer(htmlDir).ServeHTTP(w, r)
		}
	}
}
