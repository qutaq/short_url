package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestGenerateSignAndVerifyUserID(t *testing.T) {
	userID, err := GenerateUserID()
	if err != nil {
		t.Fatalf("GenerateUserID: %v", err)
	}
	if len(userID) != userIDLen*2 {
		t.Fatalf("GenerateUserID length = %d, want %d", len(userID), userIDLen*2)
	}

	token := SignUserID(userID)
	if len(token) != tokenLen {
		t.Fatalf("SignUserID token length = %d, want %d", len(token), tokenLen)
	}

	got, err := VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if got != userID {
		t.Fatalf("VerifyToken userID = %q, want %q", got, userID)
	}
}

func TestVerifyTokenRejectsInvalidInput(t *testing.T) {
	userID := strings.Repeat("a", userIDLen*2)
	token := SignUserID(userID)
	tampered := token[:len(token)-1] + "0"
	if tampered == token {
		tampered = token[:len(token)-1] + "1"
	}

	tests := []string{
		"",
		"too-short",
		tampered,
		token[:userIDLen*2] + strings.Repeat("z", sha256HexLen()),
	}

	for _, token := range tests {
		t.Run(token, func(t *testing.T) {
			_, err := VerifyToken(token)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("VerifyToken error = %v, want %v", err, ErrInvalidToken)
			}
		})
	}
}

func TestUserIDContext(t *testing.T) {
	if got, ok := UserIDFromContext(context.Background()); ok || got != "" {
		t.Fatalf("UserIDFromContext empty context = %q, %v; want empty, false", got, ok)
	}

	ctx := ContextWithUserID(t.Context(), "user-1")
	got, ok := UserIDFromContext(ctx)
	if !ok || got != "user-1" {
		t.Fatalf("UserIDFromContext = %q, %v; want user-1, true", got, ok)
	}

	ctx = ContextWithUserID(t.Context(), "")
	if got, ok := UserIDFromContext(ctx); ok || got != "" {
		t.Fatalf("UserIDFromContext empty user = %q, %v; want empty, false", got, ok)
	}
}

func sha256HexLen() int {
	return tokenLen - userIDLen*2
}
