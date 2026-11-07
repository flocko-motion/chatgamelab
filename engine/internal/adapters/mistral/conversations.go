package mistral

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// client speaks the parts of Mistral's API the adapters share.
type client struct {
	// base is a field so a test can point it at a fake server.
	base string
	http *http.Client
}

type input struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type tool struct {
	Type string `json:"type"`
}

// start is a new conversation. Store is always false: every call is one-shot,
// so nothing needs to be kept on Mistral's side.
type start struct {
	Model        string  `json:"model"`
	Instructions string  `json:"instructions,omitempty"`
	Inputs       []input `json:"inputs"`
	Tools        []tool  `json:"tools,omitempty"`
	Store        bool    `json:"store"`
}

type reply struct {
	Outputs []struct {
		Role string `json:"role"`
		// Content is a string for plain text, and an array of chunks once a
		// tool has produced something.
		Content json.RawMessage `json:"content"`
	} `json:"outputs"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
}

type chunk struct {
	Type   string `json:"type"`
	Text   string `json:"text"`
	FileID string `json:"file_id"`
}

func (c *client) converse(ctx context.Context, key string, req start) (reply, error) {
	var out reply

	body, err := json.Marshal(req)
	if err != nil {
		return out, err
	}
	resp, err := c.do(ctx, key, http.MethodPost, "/conversations", body)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&out)
	return out, err
}

// file fetches a file a tool produced, such as a generated picture.
func (c *client) file(ctx context.Context, key, id string) ([]byte, error) {
	resp, err := c.do(ctx, key, http.MethodGet, "/files/"+id+"/content", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *client) do(ctx context.Context, key, method, path string, body []byte) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		return nil, fmt.Errorf("mistral %s: %s: %s", path, resp.Status, strings.TrimSpace(string(detail)))
	}
	return resp, nil
}

// text joins what the assistant wrote, in either content shape.
func (r reply) text() string {
	var sb strings.Builder
	for _, o := range r.Outputs {
		if o.Role != "assistant" || len(o.Content) == 0 {
			continue
		}
		var s string
		if json.Unmarshal(o.Content, &s) == nil {
			sb.WriteString(s)
			continue
		}
		for _, c := range r.chunks(o.Content) {
			if c.Type == "text" {
				sb.WriteString(c.Text)
			}
		}
	}
	return sb.String()
}

// fileID is the first file a tool produced.
func (r reply) fileID() string {
	for _, o := range r.Outputs {
		if o.Role != "assistant" {
			continue
		}
		for _, c := range r.chunks(o.Content) {
			if c.Type == "tool_file" && c.FileID != "" {
				return c.FileID
			}
		}
	}
	return ""
}

func (reply) chunks(raw json.RawMessage) []chunk {
	var chunks []chunk
	_ = json.Unmarshal(raw, &chunks)
	return chunks
}
