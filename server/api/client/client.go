package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func ApiGet(endpoint string, out any) error {
	url := endpointUrl(endpoint)
	fmt.Printf("GET %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	return parseResponse(resp, out)
}

func ApiPost(endpoint string, payload any, out any) error {
	url := endpointUrl(endpoint)

	fmt.Printf("POST %s\n%v\n", url, payload)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	return parseResponse(resp, out)
}

func parseResponse(resp *http.Response, out any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api error (%d): %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	return nil
}

func endpointUrl(endpoint string) string {
	port := os.Getenv("API_PORT")
	if port == "" {
		fmt.Println("missing env API_PORT - did you source the .env file?")
		os.Exit(1)
	}

	return fmt.Sprintf("http://127.0.0.1:%s/api/%s", port, strings.TrimPrefix(endpoint, "/"))

}
