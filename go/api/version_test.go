package api

import (
	"context"
	"testing"
)

func TestGetVersionReportsTheBuild(t *testing.T) {
	Version, GitCommit, BuildTime = "2.0.0", "abc123", "2026-11-08T12:00:00Z"

	got, err := GetVersionEndpoint{}.Handle(context.Background(), GetVersionReq{})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	want := GetVersionResp{Version: "2.0.0", GitCommit: "abc123", BuildTime: "2026-11-08T12:00:00Z"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
