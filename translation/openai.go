package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const responsesURL = "https://api.openai.com/v1/responses"

type usage struct {
	Input  int `json:"input_tokens"`
	Output int `json:"output_tokens"`
	Total  int `json:"total_tokens"`
}

func (u usage) add(o usage) usage {
	return usage{u.Input + o.Input, u.Output + o.Output, u.Total + o.Total}
}

var client = &http.Client{Timeout: 10 * time.Minute}

// translate sends the same fields in each source language and returns them
// translated into the named language, as a JSON object.
func translate(ctx context.Context, apiKey string, inputs []string, language string) (string, usage, error) {
	prompt := fmt.Sprintf("Translate this JSON to %s:\n\n", language)
	for i, in := range inputs {
		prompt += fmt.Sprintf("Original #%d: \n%s\n\n", i+1, in)
	}

	body, _ := json.Marshal(map[string]any{
		"model":        model,
		"instructions": instruction,
		"input":        []map[string]string{{"role": "user", "content": prompt}},
		"store":        false,
		"text":         map[string]any{"format": map[string]string{"type": "json_object"}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, responsesURL, bytes.NewReader(body))
	if err != nil {
		return "", usage{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", usage{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", usage{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", usage{}, fmt.Errorf("OpenAI returned %d: %s", resp.StatusCode, raw)
	}

	var r struct {
		Status            string `json:"status"`
		Error             *struct{ Message string }
		IncompleteDetails *struct{ Reason string } `json:"incomplete_details"`
		Usage             usage                    `json:"usage"`
		Output            []struct {
			Type    string `json:"type"`
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", usage{}, err
	}
	if r.Error != nil {
		return "", r.Usage, fmt.Errorf("OpenAI: %s", r.Error.Message)
	}
	if r.Status != "completed" {
		reason := r.Status
		if r.IncompleteDetails != nil {
			reason += ": " + r.IncompleteDetails.Reason
		}
		return "", r.Usage, fmt.Errorf("response not completed: %s", reason)
	}
	for _, o := range r.Output {
		if o.Type != "message" || o.Role != "assistant" {
			continue
		}
		for _, c := range o.Content {
			if c.Type == "output_text" {
				if _, err := parse(c.Text); err != nil {
					return "", r.Usage, err
				}
				return c.Text, r.Usage, nil
			}
		}
	}
	return "", r.Usage, fmt.Errorf("no text in the response")
}
