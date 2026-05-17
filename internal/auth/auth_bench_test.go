package auth

import (
	"context"
	"testing"
)

func BenchmarkGenerateUserID(b *testing.B) {
	for b.Loop() {
		userID, err := GenerateUserID()
		if err != nil {
			b.Fatal(err)
		}
		if userID == "" {
			b.Fatal("empty user id")
		}
	}
}

func BenchmarkSignUserID(b *testing.B) {
	userID := "0123456789abcdef0123456789abcdef"

	for b.Loop() {
		token := SignUserID(userID)
		if token == "" {
			b.Fatal("empty token")
		}
	}
}

func BenchmarkVerifyToken(b *testing.B) {
	userID := "0123456789abcdef0123456789abcdef"
	token := SignUserID(userID)

	for b.Loop() {
		got, err := VerifyToken(token)
		if err != nil {
			b.Fatal(err)
		}
		if got != userID {
			b.Fatalf("user id = %q, want %q", got, userID)
		}
	}
}

func BenchmarkContextWithUserID(b *testing.B) {
	userID := "0123456789abcdef0123456789abcdef"
	ctx := context.Background()

	for b.Loop() {
		if _, ok := UserIDFromContext(ContextWithUserID(ctx, userID)); !ok {
			b.Fatal("missing user id")
		}
	}
}
