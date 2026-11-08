// Package client provides HTTP client functionality for CLI commands to communicate
// with the ChatGameLab backend API. This is NOT for external API calls.
//
// Features:
// - JWT token management for authentication
// - Endpoints for game, user, and API key management
// - Uses config module for server URL and JWT storage
//
// For external API calls (OpenAI, Mistral, etc.), use the apiclient package instead.
//
// package: client / CLI-to-backend HTTP client
// type:    glue
// job:     wraps HTTP verbs, JWT auth, and SSE streaming for CLI calls to the CGL API
// limits:  not for external provider calls (-> apiclient); no route handling (-> api/routes)
package client

import (
	"cgl/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// SaveJwt saves the JWT token to config (deprecated, use config.SetServerConfig)
func SaveJwt(token string) error {
	serverURL, err := config.GetServerURL()
	if err != nil {
		return fmt.Errorf("no server configured: %w", err)
	}
	return config.SetServerConfig(serverURL, token)
}

// LoadJwt loads the JWT token from config
func LoadJwt() string {
	jwt, _ := config.GetJWT()
	return jwt
}

// ApiGet performs a GET request to the given endpoint and decodes the JSON response into out.
func ApiGet(endpoint string, out any) error {
	return apiRequest("GET", endpoint, nil, out)
}

// ApiPost performs a POST request with a JSON payload and decodes the JSON response into out.
func ApiPost(endpoint string, payload any, out any) error {
	return apiRequest("POST", endpoint, payload, out)
}

// ApiDelete performs a DELETE request to the given endpoint.
func ApiDelete(endpoint string) error {
	return apiRequest("DELETE", endpoint, nil, nil)
}

// ApiPatch performs a PATCH request with a JSON payload and decodes the JSON response into out.
func ApiPatch(endpoint string, payload any, out any) error {
	return apiRequest("PATCH", endpoint, payload, out)
}

// ApiGetRaw fetches raw text content (e.g., YAML)
func ApiGetRaw(endpoint string, out *string) error {
	return apiRequestRaw("GET", endpoint, "", "", out)
}

// ApiPutRaw sends raw text content (e.g., YAML)
func ApiPutRaw(endpoint string, content string) error {
	return apiRequestRaw("PUT", endpoint, content, "application/x-yaml", nil)
}

// ApiPostRaw sends raw text content and parses JSON response
func ApiPostRaw(endpoint string, content string, out any) error {
	var rawOut string
	if err := apiRequestRaw("POST", endpoint, content, "application/x-yaml", &rawOut); err != nil {
		return err
	}
	if out != nil && rawOut != "" {
		if err := json.Unmarshal([]byte(rawOut), out); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}
	}
	return nil
}

func apiRequest(method, endpoint string, payload any, out any) error {
	var content string
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %v", err)
		}
		content = string(body)
	}

	var rawOut string
	if err := apiRequestRaw(method, endpoint, content, "application/json", &rawOut); err != nil {
		return err
	}

	if out != nil && rawOut != "" {
		if err := json.Unmarshal([]byte(rawOut), out); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}
	}
	return nil
}

func endpointUrl(endpoint string) string {
	url, err := config.GetServerURL()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return fmt.Sprintf("%s/api/%s", url, strings.TrimPrefix(endpoint, "/"))
}

func apiRequestRaw(method, endpoint string, content string, contentType string, out *string) error {
	endpoint = strings.TrimPrefix(endpoint, "/")
	url := endpointUrl(endpoint)
	fmt.Fprintf(os.Stderr, "%s %s\n", method, url)

	var bodyReader io.Reader
	if content != "" {
		bodyReader = strings.NewReader(content)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	if content != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Add JWT auth if available - except when generating jwt tokens
	if !strings.HasPrefix(endpoint, "user/jwt") {
		if jwt := LoadJwt(); jwt != "" {
			req.Header.Set("Authorization", "Bearer "+jwt)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api error (%d): %s", resp.StatusCode, string(body))
	}

	if out != nil {
		*out = string(body)
	}

	return nil
}
