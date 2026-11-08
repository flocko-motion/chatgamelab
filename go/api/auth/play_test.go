package auth

import (
	"testing"
)

func TestPlayTokenRoundTrip(t *testing.T) {
	t.Setenv("PLAY_TOKEN_SECRET", "play-secret")
	InitPlayTokens()
	token, expiresAt, err := GeneratePlayToken("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if expiresAt.IsZero() {
		t.Error("no expiry")
	}
	if sub, ok := ValidatePlayToken(token); !ok || sub != "user-1" {
		t.Errorf("ValidatePlayToken = %q, %v", sub, ok)
	}
}

// A dev token, signed with another secret and for another audience, plays nothing.
func TestPlayTokenRefusesOtherTokens(t *testing.T) {
	t.Setenv("PLAY_TOKEN_SECRET", "play-secret")
	t.Setenv("DEV_JWT_SECRET", "play-secret")
	InitPlayTokens()
	InitJwtGeneration()
	dev, _, err := GenerateToken("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ValidatePlayToken(dev); ok {
		t.Error("a dev token passed as a play token")
	}
	if _, ok := ValidatePlayToken("garbage"); ok {
		t.Error("garbage passed as a play token")
	}
}
