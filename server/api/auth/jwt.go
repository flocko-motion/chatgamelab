package auth

import (
	"cgl/functional"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// secret is used for signing/validating JWTs
var secret []byte

func InitJwtGeneration() {
	secret = []byte(functional.RequireEnv("JWT_SECRET"))
}

// GenerateToken creates a new JWT token for the given subject (user ID)
func GenerateToken(userId string) (string, int64, error) {
	expireAt := time.Now().Add(24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userId,
		"iss":   "cgl",
		"aud":   "cgl",
		"exp":   expireAt.Unix(),
		"iat":   time.Now().Unix(),
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

	log.Printf("[CGL JWT] Validating token: %s...", tokenString[:min(50, len(tokenString))])

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Printf("[CGL JWT] Invalid signing method: %v", token.Method)
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})
	if err != nil {
		log.Printf("[CGL JWT] Parse error: %v", err)
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
