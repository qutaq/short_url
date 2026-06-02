package service

import (
	"fmt"
	"testing"

	"github.com/qutaq/short_url/internal/repository"
)

const benchmarkBaseURL = "http://localhost:8080"

func BenchmarkShortenerShorten(b *testing.B) {
	ctx := b.Context()
	shortener := NewShortener(repository.NewMemoryRepository(), benchmarkBaseURL)

	for i := 0; b.Loop(); i++ {
		b.StopTimer()
		rawURL := fmt.Sprintf("https://example.com/%d", i)
		b.StartTimer()
		shortURL, err := shortener.Shorten(ctx, rawURL, "user-1")
		if err != nil {
			b.Fatal(err)
		}
		if shortURL == "" {
			b.Fatal("empty short url")
		}
	}
}

func BenchmarkShortenerShortenBatch(b *testing.B) {
	ctx := b.Context()
	shortener := NewShortener(repository.NewMemoryRepository(), benchmarkBaseURL)
	const batchSize = 100

	for batch := 0; b.Loop(); batch++ {
		b.StopTimer()
		items := make([]BatchInput, batchSize)
		for i := range items {
			n := batch*batchSize + i
			items[i] = BatchInput{
				CorrelationID: fmt.Sprintf("corr-%d", n),
				OriginalURL:   fmt.Sprintf("https://example.com/%d", n),
			}
		}
		b.StartTimer()
		results, err := shortener.ShortenBatch(ctx, items, "user-1")
		if err != nil {
			b.Fatal(err)
		}
		if len(results) != batchSize {
			b.Fatalf("results = %d, want %d", len(results), batchSize)
		}
	}
}

func BenchmarkShortenerGetOriginal(b *testing.B) {
	ctx := b.Context()
	repo := repository.NewMemoryRepository()
	shortener := NewShortener(repo, benchmarkBaseURL)
	const id = "known-id"
	const rawURL = "https://example.com/known"
	if err := repo.Save(ctx, id, rawURL, "user-1"); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		got, err := shortener.GetOriginal(ctx, id)
		if err != nil {
			b.Fatal(err)
		}
		if got != rawURL {
			b.Fatalf("url = %q, want %q", got, rawURL)
		}
	}
}

func BenchmarkShortenerGetUserURLs(b *testing.B) {
	ctx := b.Context()
	repo := repository.NewMemoryRepository()
	shortener := NewShortener(repo, benchmarkBaseURL)
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
		got, err := shortener.GetUserURLs(ctx, userID)
		if err != nil {
			b.Fatal(err)
		}
		if len(got) != urls {
			b.Fatalf("urls = %d, want %d", len(got), urls)
		}
	}
}
