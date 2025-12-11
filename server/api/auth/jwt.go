package auth

import (
	"crypto/rand"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// Secret is generated on startup and used for signing/validating JWTs
var Secret []byte

func InitJwtGeneration() {
	Secret = make([]byte, 32)
	if _, err := rand.Read(Secret); err != nil {
		log.Fatalf("Failed to generate JWT secret: %v", err)
	}
	log.Println("Generated JWT secret")
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

	tokenString, err := token.SignedString(Secret)
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

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return Secret, nil
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
