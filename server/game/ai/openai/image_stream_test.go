package openai

import (
	"cgl/game/stream"
	"cgl/obj"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func runImageStream(t *testing.T, body string) ([]byte, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	old := imageBaseURL
	imageBaseURL = srv.URL
	defer func() { imageBaseURL = old }()

	id := uuid.New()
	s := stream.Get().Create(context.Background(), &obj.GameSessionMessage{ID: id}, nil, nil)
	defer stream.Get().Remove(id)
	return callImageGenerationAPI(context.Background(), "key", "m", "low", 1, "p", id, s)
}

func sse(typ, b64 string) string {
	return `data: {"type":"` + typ + `","b64_json":"` + b64 + `"}` + "\n\n"
}

func TestImageStreamNormal(t *testing.T) {
	b := base64.StdEncoding.EncodeToString([]byte("img"))
	got, err := runImageStream(t, sse("image_generation.partial_image", b)+sse("image_generation.completed", b))
	if err != nil || string(got) != "img" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestImageStreamWithoutCompleted(t *testing.T) {
	b := base64.StdEncoding.EncodeToString([]byte("img"))
	_, err := runImageStream(t, sse("image_generation.partial_image", b))
	if err == nil || !errors.Is(err, obj.ErrImageStreamIncomplete) {
		t.Fatalf("expected incomplete error, got %v", err)
	}
}

func TestImageStreamErrorEvent(t *testing.T) {
	_, err := runImageStream(t, `data: {"type":"error","error":{"code":"rate_limit_exceeded","message":"slow down"}}`+"\n\n")
	if err == nil || !strings.Contains(err.Error(), "slow down") {
		t.Fatalf("expected error event message, got %v", err)
	}
	if errors.Is(err, obj.ErrImageStreamIncomplete) {
		t.Fatal("provider error events must not be retried as incomplete streams")
	}
	if extractImageErrorCode(err) != obj.ErrCodeRateLimitExceeded {
		t.Fatalf("wrong code %s", extractImageErrorCode(err))
	}
}
