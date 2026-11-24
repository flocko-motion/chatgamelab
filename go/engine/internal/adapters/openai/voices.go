package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"engine/internal/adapters"
)

// VoicesURL is where custom voices are made and deleted; a variable so a test
// can point it at a fake.
var VoicesURL = "https://api.openai.com/v1/audio/voices"

// Voices makes GPT-Live voices from a description (prompt-to-voice): free to
// make, used by id, at most 100 to an organisation, adult voices only.
type Voices struct {
	keys   adapters.KeyFunc
	client *http.Client
}

func NewVoices(keys adapters.KeyFunc) *Voices {
	// Making a voice is one synchronous call; give it room, it runs before a
	// game starts, not while someone talks.
	return &Voices{keys: keys, client: &http.Client{Timeout: 60 * time.Second}}
}

// CustomVoice reports whether a voice is one made from a description, by id,
// rather than a built-in voice, by name.
func CustomVoice(voice string) bool { return strings.HasPrefix(voice, "voice_") }

// Create makes a voice from a description, which the provider takes up to
// 2,000 characters of.
func (v *Voices) Create(ctx context.Context, name, description string) (string, error) {
	key, err := v.keys(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve api key: %w", err)
	}
	if len([]rune(description)) > 2000 {
		description = string([]rune(description)[:2000])
	}
	body, err := json.Marshal(map[string]any{"type": "prompt", "name": name, "prompt": description})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, VoicesURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.client.Do(req)
	if err != nil {
		return "", adapters.Transient("voices api", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		detail, _ := readError(resp)
		return "", adapters.StatusError("voices api", resp, detail)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("voices api: %w", err)
	}
	if out.ID == "" {
		return "", fmt.Errorf("voices api: no voice id in the answer")
	}
	return out.ID, nil
}

// Delete removes a voice; one already gone counts as deleted.
func (v *Voices) Delete(ctx context.Context, id string) error {
	key, err := v.keys(ctx)
	if err != nil {
		return fmt.Errorf("resolve api key: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, VoicesURL+"/"+id, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := v.client.Do(req)
	if err != nil {
		return adapters.Transient("voices api", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
		return nil
	}
	detail, _ := readError(resp)
	return adapters.StatusError("voices api", resp, detail)
}

// listPages bounds a listing: an organisation holds at most 100 voices.
const listPages = 10

// List answers the voices the account holds, page by page. The provider
// documents no list, though it answers one; a failure says nothing about the
// voices, so a caller goes on without it.
func (v *Voices) List(ctx context.Context) ([]adapters.VoiceInfo, error) {
	key, err := v.keys(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve api key: %w", err)
	}
	var all []adapters.VoiceInfo
	after := ""
	for page := 0; page < listPages; page++ {
		url := VoicesURL + "?limit=100"
		if after != "" {
			url += "&after=" + after
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := v.client.Do(req)
		if err != nil {
			return nil, adapters.Transient("voices api", err)
		}
		var out struct {
			Data []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				CreatedAt int64  `json:"created_at"`
			} `json:"data"`
			LastID  string `json:"last_id"`
			HasMore bool   `json:"has_more"`
		}
		if resp.StatusCode != http.StatusOK {
			detail, _ := readError(resp)
			resp.Body.Close()
			return nil, adapters.StatusError("voices api", resp, detail)
		}
		err = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("voices api: %w", err)
		}
		for _, d := range out.Data {
			all = append(all, adapters.VoiceInfo{ID: d.ID, Name: d.Name, CreatedAt: time.Unix(d.CreatedAt, 0)})
		}
		if !out.HasMore || out.LastID == "" {
			return all, nil
		}
		after = out.LastID
	}
	return all, nil
}
