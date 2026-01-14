// Package client provides HTTP client functionality for CLI commands to communicate
// with the ChatGameLab backend API. This is NOT for external API calls.
//
// Features:
// - JWT token management for authentication
// - Endpoints for game, user, and API key management
// - SSE streaming support for game sessions
// - Uses config module for server URL and JWT storage
//
// For external API calls (OpenAI, Mistral, etc.), use the apiclient package instead.
package client

import (
	"bufio"
	"cgl/config"
	"cgl/obj"
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

// GetJwtPath returns a description of where JWT is stored (for display purposes)
func GetJwtPath() string {
	configPath, _ := config.GetConfigPath()
	return configPath + " (server.jwt)"
}

func ApiGet(endpoint string, out any) error {
	return apiRequest("GET", endpoint, nil, out)
}

func ApiPost(endpoint string, payload any, out any) error {
	return apiRequest("POST", endpoint, payload, out)
}

func ApiDelete(endpoint string) error {
	return apiRequest("DELETE", endpoint, nil, nil)
}

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

// StreamSSE connects to an SSE endpoint and calls the handler for each chunk
// Returns when the stream is complete (both textDone and imageDone) or on error
func StreamSSE(endpoint string, handler func(chunk obj.GameSessionMessageChunk) error) error {
	url := endpointUrl(endpoint)
	fmt.Fprintf(os.Stderr, "SSE %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Accept", "text/event-stream")
	if jwt := LoadJwt(); jwt != "" {
		req.Header.Set("Authorization", "Bearer "+jwt)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SSE error (%d): %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer size to handle large image chunks (up to 10MB)
	const maxScanTokenSize = 10 * 1024 * 1024
	scanner.Buffer(make([]byte, 64*1024), maxScanTokenSize)
	textDone := false
	imageDone := false

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var chunk obj.GameSessionMessageChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if err := handler(chunk); err != nil {
			return err
		}

		if chunk.TextDone {
			textDone = true
		}
		if chunk.ImageDone {
			imageDone = true
		}
		if chunk.Error != "" {
			return fmt.Errorf("stream error: %s", chunk.Error)
		}
		if textDone && imageDone {
			break
		}
	}

	return scanner.Err()
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
