package repository

import (
	"fmt"
	"testing"
)

func BenchmarkMemoryRepositorySave(b *testing.B) {
	ctx := b.Context()
	repo := NewMemoryRepository()

	for i := 0; b.Loop(); i++ {
		id := fmt.Sprintf("id-%d", i)
		rawURL := fmt.Sprintf("https://example.com/%d", i)
		if err := repo.Save(ctx, id, rawURL, "user-1"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryRepositoryGet(b *testing.B) {
	ctx := b.Context()
	repo := NewMemoryRepository()
	const id = "known-id"
	const rawURL = "https://example.com/known"
	if err := repo.Save(ctx, id, rawURL, "user-1"); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		got, err := repo.Get(ctx, id)
		if err != nil {
			b.Fatal(err)
		}
		if got != rawURL {
			b.Fatalf("url = %q, want %q", got, rawURL)
		}
	}
}

func BenchmarkMemoryRepositorySaveBatch(b *testing.B) {
	ctx := b.Context()
	repo := NewMemoryRepository()
	const batchSize = 100

	for batch := 0; b.Loop(); batch++ {
		entries := make([]BatchEntry, batchSize)
		for i := range entries {
			n := batch*batchSize + i
			entries[i] = BatchEntry{
				ID:     fmt.Sprintf("id-%d", n),
				URL:    fmt.Sprintf("https://example.com/%d", n),
				UserID: "user-1",
			}
		}
		if err := repo.SaveBatch(ctx, entries); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryRepositoryGetURLsByUser(b *testing.B) {
	ctx := b.Context()
	repo := NewMemoryRepository()
	const userID = "user-1"
	const urls = 1000
	for i := 0; i < urls; i++ {
		id := fmt.Sprintf("id-%d", i)
		rawURL := fmt.Sprintf("https://example.com/%d", i)
		if err := repo.Save(ctx, id, rawURL, userID); err != nil {
			b.Fatal(err)
		}
	}

	for b.Loop() {
		pairs, err := repo.GetURLsByUser(ctx, userID)
		if err != nil {
			b.Fatal(err)
		}
		if len(pairs) != urls {
			b.Fatalf("urls = %d, want %d", len(pairs), urls)
		}
	}
}
