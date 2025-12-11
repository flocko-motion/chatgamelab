package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"cgl/api"
	"cgl/db"
	"cgl/endpoints"

	"github.com/spf13/cobra"
)

var devMode bool

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the CGL server",
	Long:  "Start the Chat Game Lab HTTP server.",
	Run:   runServer,
}

func init() {
	serverCmd.Flags().BoolVar(&devMode, "dev", false, "Enable development mode")
	rootCmd.AddCommand(serverCmd)
}

func runServer(cmd *cobra.Command, args []string) {
	endpoints.DevMode = devMode

	if devMode {
		log.Println("Development mode enabled")
	}

	db.Init()

	router := api.NewRouter([]api.Endpoint{
		endpoints.Game,
		endpoints.Games,
		endpoints.Image,
		endpoints.Report,
		endpoints.Session,
		endpoints.Status,
		endpoints.Upgrade,
		endpoints.User,
		endpoints.Version,
		endpoints.PublicGame,
		endpoints.PublicSession,
	})

	htmlDir := http.Dir("./html")
	router.Handle("/", api.SpaHandler(htmlDir, "./html/index.html"))

	http.Handle("/", api.CorsMiddleware(router))

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server listening on http://localhost:%s\n", port)
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", port), nil); err != nil {
		log.Fatalf("There was an error with the http server: %v", err)
	}
}
