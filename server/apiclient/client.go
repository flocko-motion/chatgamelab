// Package apiclient provides a generic, reusable HTTP client for calling external APIs.
// This is used for AI platform integrations (OpenAI, Mistral, etc.).
//
// Features:
// - Configurable base URL and headers
// - Context-aware requests
// - JSON marshaling/unmarshaling helpers
// - Reusable client instances
//
// For ChatGameLab backend API calls from CLI commands, use the api/client package instead.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client represents a reusable HTTP client for API calls
type Client struct {
	baseURL string
	headers map[string]string
	client  *http.Client
}

// NewApi creates a new API client with the given base URL and headers
func NewApi(baseURL string, headers map[string]string) *Client {
	// Set default headers if not provided
	if headers == nil {
		headers = make(map[string]string)
	}
	if _, exists := headers["Content-Type"]; !exists {
		headers["Content-Type"] = "application/json"
	}

	return &Client{
		baseURL: baseURL,
		headers: headers,
		client:  &http.Client{},
	}
}

// Get makes a GET request and returns the raw response
func (c *Client) Get(ctx context.Context, path string) (*Response, error) {
	return c.doRequest(ctx, "GET", path, nil)
}

// Post makes a POST request with raw body and returns the raw response
func (c *Client) Post(ctx context.Context, path string, body []byte) (*Response, error) {
	return c.doRequest(ctx, "POST", path, body)
}

// GetJson makes a GET request and unmarshals the response into the provided struct
func (c *Client) GetJson(ctx context.Context, path string, response interface{}) error {
	resp, err := c.Get(ctx, path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(response)
}

// PostJson marshals the request body, makes a POST request, and unmarshals the response
func (c *Client) PostJson(ctx context.Context, path string, request, response interface{}) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.Post(ctx, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(response)
}

// SetHeader sets or updates a header
func (c *Client) SetHeader(key, value string) {
	if c.headers == nil {
		c.headers = make(map[string]string)
	}
	c.headers[key] = value
}

// doRequest is the internal method that performs the actual HTTP request
func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) (*Response, error) {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewBuffer(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return &Response{Response: resp}, nil
}

// Response wraps http.Response to provide additional helper methods
type Response struct {
	*http.Response
}

// ReadAll reads and returns the entire response body
func (r *Response) ReadAll() ([]byte, error) {
	if r.Body == nil {
		return nil, fmt.Errorf("response body is nil")
	}
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}
