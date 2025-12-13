package client

import (
	"bufio"
	"cgl/functional"
	"cgl/obj"
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
	url := functional.RequireEnv("PUBLIC_URL")
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
