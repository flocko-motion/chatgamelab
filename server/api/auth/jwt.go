package auth

import (
	"cgl/functional"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// secret is used for signing/validating JWTs
var secret []byte

func InitJwtGeneration() {
	secret = []byte(functional.EnvOrDefault("DEV_JWT_SECRET", ""))
}

// GenerateToken creates a new JWT token for the given subject (user ID)
func GenerateToken(userId string) (string, int64, error) {
	if len(secret) == 0 {
		return "", 0, fmt.Errorf("DEV_JWT_SECRET not set - dev JWT generation disabled")
	}

	now := time.Now()
	expireAt := now.Add(24 * time.Hour)
	// Set iat 5 seconds in the past to handle clock skew and race conditions
	issuedAt := now.Add(-5 * time.Second)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userId,
		"iss":   "cgl",
		"aud":   "cgl",
		"exp":   expireAt.Unix(),
		"iat":   issuedAt.Unix(),
		"scope": "openid profile email",
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", 0, err
	}

	return tokenString, expireAt.Unix(), nil
}

// ValidateToken checks if the request has a valid CGL JWT token
// Returns the userId from the token if valid, empty string otherwise
func ValidateToken(r *http.Request) (userId string, valid bool) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", false
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	return ValidateTokenString(tokenString)
}

// ValidateTokenString checks if the given token string is a valid CGL JWT token
// Returns the userId from the token if valid, empty string otherwise
func ValidateTokenString(tokenString string) (userId string, valid bool) {
	if len(secret) == 0 {
		return "", false
	}

	if tokenString == "" {
		return "", false
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})
	if err != nil {
		return "", false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", false
	}

	// Check issuer is our issuer
	if iss, ok := claims["iss"].(string); !ok || iss != "cgl" {
		return "", false
	}

	// Extract userId from subject claim
	sub, ok := claims["sub"].(string)
	if !ok {
		return "", false
	}

	return sub, true
}
