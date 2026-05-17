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
const tokenLen = userIDLen*2 + sha256.Size*2

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

	var digest [sha256.Size]byte
	signature := mac.Sum(digest[:0])
	token := make([]byte, len(userID)+hex.EncodedLen(len(signature)))
	copy(token, userID)
	hex.Encode(token[len(userID):], signature)
	return string(token)
}

func VerifyToken(token string) (string, error) {
	if len(token) != tokenLen {
		return "", ErrInvalidToken
	}
	userID := token[:userIDLen*2]
	sigHex := token[userIDLen*2:]

	var sig [sha256.Size]byte
	if _, err := hex.Decode(sig[:], []byte(sigHex)); err != nil {
		return "", ErrInvalidToken
	}

	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(userID))
	var digest [sha256.Size]byte
	expected := mac.Sum(digest[:0])

	if !hmac.Equal(sig[:], expected) {
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
