package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

type contextKey string

const userIDKey contextKey = "user_id"

var secretKey = []byte("short-url-secret-key-2024")

const userIDLen = 16

var (
	ErrInvalidToken = errors.New("invalid auth token")
	ErrNoUserID     = errors.New("no user id in context")
)

func GenerateUserID() (string, error) {
	b := make([]byte, userIDLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate user id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func SignUserID(userID string) string {
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(userID))
	signature := mac.Sum(nil)
	return userID + hex.EncodeToString(signature)
}

func VerifyToken(token string) (string, error) {
	if len(token) < userIDLen*2+sha256.Size*2 {
		return "", ErrInvalidToken
	}
	userID := token[:userIDLen*2]
	sigHex := token[userIDLen*2:]

	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return "", ErrInvalidToken
	}

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(userID))
	expected := mac.Sum(nil)

	if !hmac.Equal(sig, expected) {
		return "", ErrInvalidToken
	}
	return userID, nil
}

func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	uid, ok := ctx.Value(userIDKey).(string)
	return uid, ok && uid != ""
}
