package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qutaq/short_url/internal/auth"
)

func TestAuthMiddlewareCreatesCookieAndContext(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("user id missing from request context")
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(userID))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	AuthMiddleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies len = %d, want 1", len(cookies))
	}
	if cookies[0].Name != cookieName || cookies[0].Value == "" || !cookies[0].HttpOnly {
		t.Fatalf("cookie = %#v, want auth cookie with HttpOnly token", cookies[0])
	}

	userID, err := auth.VerifyToken(cookies[0].Value)
	if err != nil {
		t.Fatalf("VerifyToken cookie: %v", err)
	}
	if rec.Body.String() != userID {
		t.Fatalf("context user id = %q, want %q", rec.Body.String(), userID)
	}
}

func TestAuthMiddlewareAcceptsExistingCookie(t *testing.T) {
	userID := "12345678901234567890123456789012"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, ok := auth.UserIDFromContext(r.Context())
		if !ok || got != userID {
			t.Fatalf("context user id = %q, %v; want %q, true", got, ok, userID)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: auth.SignUserID(userID)})
	rec := httptest.NewRecorder()
	AuthMiddleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Result().Cookies(); len(got) != 0 {
		t.Fatalf("new cookies = %#v, want none", got)
	}
}

func TestGzipMiddlewareCompressesResponse(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	GzipMiddleware(next).ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", rec.Header().Get("Content-Encoding"))
	}

	gz, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("NewReader response: %v", err)
	}
	defer gz.Close()
	body, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("ReadAll response: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("decompressed body = %q, want JSON", body)
	}
}

func TestGzipMiddlewareDecompressesRequest(t *testing.T) {
	var body bytes.Buffer
	gz := gzip.NewWriter(&body)
	if _, err := gz.Write([]byte("payload")); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll request: %v", err)
		}
		if string(got) != "payload" {
			t.Fatalf("request body = %q, want payload", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()
	GzipMiddleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestGzipMiddlewareRejectsInvalidGzipRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("not gzip"))
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	GzipMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
