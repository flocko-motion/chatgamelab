// package: auth / play tokens for the playback-engine
// type:    logic
// job:     generates and validates the short-lived tokens that let the playback-engine act for a player
// limits:  no cookie handling (-> httpx), no user lookup; signed with PLAY_TOKEN_SECRET, never the dev secret
package auth

import (
	"crypto/rand"
	"fmt"
	"time"

	"cgl/functional"
	"cgl/log"

	"github.com/golang-jwt/jwt/v4"
)

// PlayTokenLifetime is how long a play token holds; the web app renews it
// while a game page is open.
const PlayTokenLifetime = time.Hour

const (
	playIssuer   = "cgl-play"
	playAudience = "cgl-engine"
)

// playSecret signs play tokens; set by InitPlayTokens.
var playSecret []byte

// InitPlayTokens loads PLAY_TOKEN_SECRET. Without it a random secret holds
// for this process, so play tokens lapse on restart and the web app renews them.
func InitPlayTokens() {
	playSecret = []byte(functional.EnvOrDefault("PLAY_TOKEN_SECRET", ""))
	if len(playSecret) > 0 {
		return
	}
	playSecret = make([]byte, 32)
	if _, err := rand.Read(playSecret); err != nil {
		panic(err)
	}
	log.Warn("PLAY_TOKEN_SECRET not set; play tokens lapse when the server restarts")
}

// GeneratePlayToken makes a play token for a user, and when it expires.
func GeneratePlayToken(userID string) (string, time.Time, error) {
	if len(playSecret) == 0 {
		return "", time.Time{}, fmt.Errorf("play tokens are not initialised")
	}
	now := time.Now()
	expiresAt := now.Add(PlayTokenLifetime)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"iss": playIssuer,
		"aud": playAudience,
		"exp": expiresAt.Unix(),
		"iat": now.Add(-5 * time.Second).Unix(),
	})
	signed, err := token.SignedString(playSecret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

// ValidatePlayToken answers the user a play token was made for, if it is
// genuine, unexpired and meant for the engine.
func ValidatePlayToken(tokenString string) (userID string, valid bool) {
	if len(playSecret) == 0 || tokenString == "" {
		return "", false
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return playSecret, nil
	})
	if err != nil || !token.Valid {
		return "", false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !claims.VerifyIssuer(playIssuer, true) || !claims.VerifyAudience(playAudience, true) {
		return "", false
	}
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", false
	}
	return sub, true
}
