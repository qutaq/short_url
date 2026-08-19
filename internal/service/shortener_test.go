package service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/qutaq/short_url/internal/repository"
)

func TestShortenCreatesURLAndReturnsExistingConflict(t *testing.T) {
	ctx := t.Context()
	repo := repository.NewMemoryRepository()
	shortener := NewShortener(repo, "http://localhost:8080")

	first, err := shortener.Shorten(ctx, "https://example.com", "user1")
	if err != nil {
		t.Fatalf("Shorten first: %v", err)
	}
	if !strings.HasPrefix(first, "http://localhost:8080/") {
		t.Fatalf("Shorten first = %q, want base URL prefix", first)
	}

	second, err := shortener.Shorten(ctx, "https://example.com", "user1")
	if !errors.Is(err, ErrURLConflict) {
		t.Fatalf("Shorten duplicate error = %v, want %v", err, ErrURLConflict)
	}
	if second != first {
		t.Fatalf("Shorten duplicate = %q, want existing %q", second, first)
	}
}

func TestShortenRejectsInvalidInput(t *testing.T) {
	ctx := t.Context()
	shortener := NewShortener(repository.NewMemoryRepository(), "http://localhost:8080")

	got, err := shortener.Shorten(ctx, "", "user1")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Shorten empty error = %v, want %v", err, ErrInvalidInput)
	}
	if got != "" {
		t.Fatalf("Shorten empty result = %q, want empty", got)
	}
}

func TestShortenBatchAndGetUserURLs(t *testing.T) {
	ctx := t.Context()
	shortener := NewShortener(repository.NewMemoryRepository(), "http://localhost:8080")

	got, err := shortener.ShortenBatch(ctx, []BatchInput{
		{CorrelationID: "1", OriginalURL: "https://example.com/1"},
		{CorrelationID: "2", OriginalURL: "https://example.com/2"},
	}, "user1")
	if err != nil {
		t.Fatalf("ShortenBatch: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ShortenBatch len = %d, want 2", len(got))
	}
	for i, item := range got {
		if item.CorrelationID == "" {
			t.Fatalf("ShortenBatch item %d has empty correlation id", i)
		}
		if !strings.HasPrefix(item.ShortURL, "http://localhost:8080/") {
			t.Fatalf("ShortenBatch item %d ShortURL = %q, want base URL prefix", i, item.ShortURL)
		}
	}

	userURLs, err := shortener.GetUserURLs(ctx, "user1")
	if err != nil {
		t.Fatalf("GetUserURLs: %v", err)
	}
	if len(userURLs) != 2 {
		t.Fatalf("GetUserURLs len = %d, want 2", len(userURLs))
	}
}

func TestShortenBatchRejectsInvalidInput(t *testing.T) {
	ctx := t.Context()
	shortener := NewShortener(repository.NewMemoryRepository(), "http://localhost:8080")

	if got, err := shortener.ShortenBatch(ctx, nil, "user1"); !errors.Is(err, ErrInvalidInput) || got != nil {
		t.Fatalf("ShortenBatch empty = %#v, %v; want nil, %v", got, err, ErrInvalidInput)
	}
	if got, err := shortener.ShortenBatch(ctx, []BatchInput{{CorrelationID: "1"}}, "user1"); !errors.Is(err, ErrInvalidInput) || got != nil {
		t.Fatalf("ShortenBatch empty URL = %#v, %v; want nil, %v", got, err, ErrInvalidInput)
	}
}

func TestGetOriginalMapsRepositoryErrors(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "not found", repoErr: repository.ErrNotFound, wantErr: ErrNotFound},
		{name: "deleted", repoErr: repository.ErrDeleted, wantErr: ErrDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortener := NewShortener(&stubRepo{getErr: tt.repoErr}, "http://localhost:8080")
			got, err := shortener.GetOriginal(t.Context(), "short1")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetOriginal error = %v, want %v", err, tt.wantErr)
			}
			if got != "" {
				t.Fatalf("GetOriginal = %q, want empty", got)
			}
		})
	}
}

func TestGetUserURLsMapsRepositoryPairs(t *testing.T) {
	ctx := t.Context()
	shortener := NewShortener(&stubRepo{
		urlPairs: []repository.URLPair{
			{ShortID: "short1", OriginalURL: "https://example.com/1"},
			{ShortID: "short2", OriginalURL: "https://example.com/2"},
		},
	}, "http://localhost:8080")

	got, err := shortener.GetUserURLs(ctx, "user1")
	if err != nil {
		t.Fatalf("GetUserURLs: %v", err)
	}
	want := []UserURLOutput{
		{ShortURL: "http://localhost:8080/short1", OriginalURL: "https://example.com/1"},
		{ShortURL: "http://localhost:8080/short2", OriginalURL: "https://example.com/2"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetUserURLs = %#v, want %#v", got, want)
	}
}

func TestFlushDeleteBatchGroupsByUser(t *testing.T) {
	repo := &stubRepo{deleted: make(map[string][]string)}
	shortener := &Shortener{repo: repo}

	shortener.flushDeleteBatch([]deleteTask{
		{shortID: "a", userID: "user1"},
		{shortID: "b", userID: "user2"},
		{shortID: "c", userID: "user1"},
	})

	if !reflect.DeepEqual(repo.deleted["user1"], []string{"a", "c"}) {
		t.Fatalf("deleted for user1 = %#v, want [a c]", repo.deleted["user1"])
	}
	if !reflect.DeepEqual(repo.deleted["user2"], []string{"b"}) {
		t.Fatalf("deleted for user2 = %#v, want [b]", repo.deleted["user2"])
	}
}

func TestCloseFlushesPendingDeletes(t *testing.T) {
	repo := &stubRepo{deleted: make(map[string][]string)}
	shortener := NewShortener(repo, "http://localhost:8080")

	shortener.DeleteUserURLs([]string{"a", "b"}, "user1")
	shortener.Close()

	if !reflect.DeepEqual(repo.deleted["user1"], []string{"a", "b"}) {
		t.Fatalf("deleted for user1 = %#v, want [a b]", repo.deleted["user1"])
	}
}

func TestDeleteUserURLsAfterCloseIsIgnored(t *testing.T) {
	repo := &stubRepo{deleted: make(map[string][]string)}
	shortener := NewShortener(repo, "http://localhost:8080")

	shortener.Close()
	shortener.DeleteUserURLs([]string{"late"}, "user1")

	if len(repo.deleted) != 0 {
		t.Fatalf("deleted after close = %#v, want empty", repo.deleted)
	}
}

func TestGenerateDeleteTasks(t *testing.T) {
	got := generateDeleteTasks([]string{"a", "b"}, "user1")
	want := []deleteTask{{shortID: "a", userID: "user1"}, {shortID: "b", userID: "user1"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("generateDeleteTasks = %#v, want %#v", got, want)
	}
}

type stubRepo struct {
	getErr   error
	urlPairs []repository.URLPair
	deleted  map[string][]string
}

func (r *stubRepo) Save(context.Context, string, string, string) error {
	return nil
}

func (r *stubRepo) SaveBatch(context.Context, []repository.BatchEntry) error {
	return nil
}

func (r *stubRepo) Get(context.Context, string) (string, error) {
	if r.getErr != nil {
		return "", r.getErr
	}
	return "https://example.com", nil
}

func (r *stubRepo) GetByOriginalURL(context.Context, string) (string, bool, error) {
	return "", false, nil
}

func (r *stubRepo) GetURLsByUser(context.Context, string) ([]repository.URLPair, error) {
	return r.urlPairs, nil
}

func (r *stubRepo) DeleteUserURLs(_ context.Context, shortIDs []string, userID string) error {
	r.deleted[userID] = append(r.deleted[userID], shortIDs...)
	return nil
}

func (r *stubRepo) GetStats(context.Context) (repository.Stats, error) {
	return repository.Stats{}, nil
}
