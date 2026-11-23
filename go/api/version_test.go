package api

import (
	"context"
	"testing"
)

func TestGetVersionReportsTheRunningRelease(t *testing.T) {
	for _, c := range []struct{ image, want string }{
		{"chatgamelab:2.0.0-dev.7", "2.0.0-dev.7"},
		{"", "dev"}, // development: no release runs
	} {
		t.Setenv("SCHEMAF_IMAGE", c.image)
		got, err := GetVersionEndpoint{}.Handle(context.Background(), GetVersionReq{})
		if err != nil {
			t.Fatalf("Handle: %v", err)
		}
		if got.Version != c.want {
			t.Errorf("image %q: got %q, want %q", c.image, got.Version, c.want)
		}
	}
}
