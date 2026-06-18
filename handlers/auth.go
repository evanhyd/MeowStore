package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	jwtSecret []byte
}

func NewAuthHandler(secret []byte) *AuthHandler {
	return &AuthHandler{jwtSecret: secret}
}

// verifyToken checks the JWT signature and extracts the user id.
func (h *AuthHandler) verifyToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims structure")
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return "", errors.New("user_id (sub) missing from token")
	}

	return userID, nil
}

// Helper to extract and verify the JWT from the Authorization header
func (h *AuthHandler) getUserIDFromHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header format")
	}

	return h.verifyToken(parts[1])
}
