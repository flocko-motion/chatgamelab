package client

import (
	"bytes"
	"cgl/functional"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var cglDir string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Warning: could not get home directory: %v\n", err)
		return
	}
	cglDir = filepath.Join(home, ".cgl")
}

// GetJwtPath returns the path to the JWT file
func GetJwtPath() string {
	return filepath.Join(cglDir, "jwt")
}

// SaveJwt saves the JWT token to ~/.cgl/jwt
func SaveJwt(token string) error {
	if err := os.MkdirAll(cglDir, 0700); err != nil {
		return fmt.Errorf("failed to create %s: %v", cglDir, err)
	}
	if err := os.WriteFile(GetJwtPath(), []byte(token), 0600); err != nil {
		return fmt.Errorf("failed to write JWT: %v", err)
	}
	return nil
}

// LoadJwt loads the JWT token from ~/.cgl/jwt
func LoadJwt() string {
	data, err := os.ReadFile(GetJwtPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func ApiGet(endpoint string, out any) error {
	return apiRequest("GET", endpoint, nil, out)
}

func ApiPost(endpoint string, payload any, out any) error {
	return apiRequest("POST", endpoint, payload, out)
}

func apiRequest(method, endpoint string, payload any, out any) error {
	endpoint = strings.TrimPrefix(endpoint, "/")
	url := endpointUrl(endpoint)
	fmt.Printf("%s %s\n", method, url)

	var bodyReader io.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %v", err)
		}
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add JWT auth if available - except when trying to generate dev jwt tokens
	if !strings.HasPrefix(endpoint, "user/jwt") {
		if jwt := LoadJwt(); jwt != "" {
			req.Header.Set("Authorization", "Bearer "+jwt)
			fmt.Printf("Authorization %s..\n", req.Header.Get("Authorization")[0:30])
		}
	}

	resp, err := http.DefaultClient.Do(req)
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
	url := functional.RequireEnv("PUBLIC_URL")
	return fmt.Sprintf("%s/api/%s", url, strings.TrimPrefix(endpoint, "/"))

}
