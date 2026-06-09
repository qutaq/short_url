package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMemoryRepositorySaveGetAndDelete(t *testing.T) {
	ctx := t.Context()
	repo := NewMemoryRepository()

	if err := repo.Save(ctx, "short1", "https://example.com/1", "user1"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := repo.Save(ctx, "short2", "https://example.com/2", "user2"); err != nil {
		t.Fatalf("Save second url: %v", err)
	}

	got, err := repo.Get(ctx, "short1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "https://example.com/1" {
		t.Fatalf("Get = %q, want original url", got)
	}

	id, ok, err := repo.GetByOriginalURL(ctx, "https://example.com/1")
	if err != nil {
		t.Fatalf("GetByOriginalURL: %v", err)
	}
	if !ok || id != "short1" {
		t.Fatalf("GetByOriginalURL = %q, %v; want short1, true", id, ok)
	}

	pairs, err := repo.GetURLsByUser(ctx, "user1")
	if err != nil {
		t.Fatalf("GetURLsByUser: %v", err)
	}
	wantPairs := []URLPair{{ShortID: "short1", OriginalURL: "https://example.com/1"}}
	if !reflect.DeepEqual(pairs, wantPairs) {
		t.Fatalf("GetURLsByUser = %#v, want %#v", pairs, wantPairs)
	}

	if err := repo.DeleteUserURLs(ctx, []string{"short1", "short2"}, "user1"); err != nil {
		t.Fatalf("DeleteUserURLs: %v", err)
	}
	if _, err := repo.Get(ctx, "short1"); !errors.Is(err, ErrDeleted) {
		t.Fatalf("Get deleted error = %v, want %v", err, ErrDeleted)
	}
	if _, err := repo.Get(ctx, "short2"); err != nil {
		t.Fatalf("Get url owned by another user: %v", err)
	}
}

func TestMemoryRepositoryConflictsAndValidation(t *testing.T) {
	ctx := t.Context()
	repo := NewMemoryRepository()

	if err := repo.Save(ctx, "short1", "https://example.com/1", "user1"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := repo.Save(ctx, "short1", "https://example.com/other", "user1"); !errors.Is(err, ErrConflict) {
		t.Fatalf("Save duplicate id error = %v, want %v", err, ErrConflict)
	}
	if err := repo.Save(ctx, "short2", "https://example.com/1", "user1"); !errors.Is(err, ErrURLExists) {
		t.Fatalf("Save duplicate url error = %v, want %v", err, ErrURLExists)
	}
	if err := repo.Save(ctx, "short3", "https://example.com/3", strings.Repeat("u", maxUserIDLen+1)); !errors.Is(err, ErrUserIDLen) {
		t.Fatalf("Save long user id error = %v, want %v", err, ErrUserIDLen)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := repo.Save(cancelled, "short4", "https://example.com/4", "user1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Save cancelled context error = %v, want %v", err, context.Canceled)
	}
}

func TestMemoryRepositorySaveBatchIsAtomic(t *testing.T) {
	ctx := t.Context()
	repo := NewMemoryRepository()

	entries := []BatchEntry{
		{ID: "short1", URL: "https://example.com/1", UserID: "user1"},
		{ID: "short2", URL: "https://example.com/2", UserID: "user1"},
	}
	if err := repo.SaveBatch(ctx, entries); err != nil {
		t.Fatalf("SaveBatch: %v", err)
	}

	if err := repo.SaveBatch(ctx, []BatchEntry{
		{ID: "short3", URL: "https://example.com/3", UserID: "user1"},
		{ID: "short2", URL: "https://example.com/4", UserID: "user1"},
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("SaveBatch conflict error = %v, want %v", err, ErrConflict)
	}

	if _, err := repo.Get(ctx, "short3"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get partially saved entry error = %v, want %v", err, ErrNotFound)
	}
}

func TestFileRepositoryPersistsRecords(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "urls.jsonl")

	repo, err := NewFileRepository(path)
	if err != nil {
		t.Fatalf("NewFileRepository: %v", err)
	}
	if err := repo.Save(ctx, "short1", "https://example.com/1", "user1"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := repo.SaveBatch(ctx, []BatchEntry{
		{ID: "short2", URL: "https://example.com/2", UserID: "user2"},
		{ID: "short3", URL: "https://example.com/3", UserID: ""},
	}); err != nil {
		t.Fatalf("SaveBatch: %v", err)
	}

	reloaded, err := NewFileRepository(path)
	if err != nil {
		t.Fatalf("NewFileRepository reload: %v", err)
	}
	got, err := reloaded.Get(ctx, "short1")
	if err != nil {
		t.Fatalf("Get reloaded: %v", err)
	}
	if got != "https://example.com/1" {
		t.Fatalf("Get reloaded = %q, want original url", got)
	}

	id, ok, err := reloaded.GetByOriginalURL(ctx, "https://example.com/2")
	if err != nil {
		t.Fatalf("GetByOriginalURL reloaded: %v", err)
	}
	if !ok || id != "short2" {
		t.Fatalf("GetByOriginalURL reloaded = %q, %v; want short2, true", id, ok)
	}
}

func TestFileRepositoryClosePersistsDeletedURLs(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "urls.jsonl")

	repo, err := NewFileRepository(path)
	if err != nil {
		t.Fatalf("NewFileRepository: %v", err)
	}
	if err := repo.Save(ctx, "short1", "https://example.com/1", "user1"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := repo.DeleteUserURLs(ctx, []string{"short1"}, "user1"); err != nil {
		t.Fatalf("DeleteUserURLs: %v", err)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reloaded, err := NewFileRepository(path)
	if err != nil {
		t.Fatalf("NewFileRepository reload: %v", err)
	}
	if _, err := reloaded.Get(ctx, "short1"); !errors.Is(err, ErrDeleted) {
		t.Fatalf("Get deleted reloaded error = %v, want %v", err, ErrDeleted)
	}
}

func TestFileRepositoryLoadErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "urls.jsonl")
	if err := os.WriteFile(path, []byte("{invalid json}\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := NewFileRepository(path); err == nil {
		t.Fatal("NewFileRepository invalid json error = nil, want error")
	}
}
