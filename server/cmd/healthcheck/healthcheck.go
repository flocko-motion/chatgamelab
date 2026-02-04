package healthcheck

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"cgl/functional"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "Check if the server is healthy",
	Long:  "Performs a health check by querying the /api/status endpoint. Exits with code 0 if healthy, 1 if unhealthy.",
	Run:   run,
}

func run(cmd *cobra.Command, args []string) {
	port := functional.EnvOrDefault("PORT_BACKEND", "3000")
	url := fmt.Sprintf("http://localhost:%s/api/status", port)

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Health check failed: status code %d\n", resp.StatusCode)
		os.Exit(1)
	}

	os.Exit(0)
}
