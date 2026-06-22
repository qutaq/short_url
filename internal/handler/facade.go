package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/qutaq/short_url/internal/audit"
	"github.com/qutaq/short_url/internal/auth"
	"github.com/qutaq/short_url/internal/service"
)

// ShortenResult описывает результат операции сокращения URL.
type ShortenResult struct {
	ShortURL string
	Conflict bool
}

// ShortenerFacade инкапсулирует общую бизнес-логику для HTTP- и gRPC-обработчиков.
type ShortenerFacade struct {
	shortener *service.Shortener
	auditor   audit.Observer
}

// NewShortenerFacade создаёт фасад поверх сервиса сокращения ссылок.
func NewShortenerFacade(shortener *service.Shortener, auditors ...audit.Observer) *ShortenerFacade {
	auditor := audit.Observer(audit.NewNotifier())
	if len(auditors) > 0 {
		auditor = auditors[0]
	}
	return &ShortenerFacade{shortener: shortener, auditor: auditor}
}

// Shortener возвращает сервис сокращения ссылок, используемый фасадом.
func (f *ShortenerFacade) Shortener() *service.Shortener {
	return f.shortener
}

// ShortenURL сокращает URL для пользователя из контекста.
func (f *ShortenerFacade) ShortenURL(ctx context.Context, rawURL string) (ShortenResult, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || !isValidURL(rawURL) {
		return ShortenResult{}, service.ErrInvalidInput
	}

	userID, _ := auth.UserIDFromContext(ctx)

	shortURL, err := f.shortener.Shorten(ctx, rawURL, userID)
	if errors.Is(err, service.ErrURLConflict) {
		return ShortenResult{ShortURL: shortURL, Conflict: true}, nil
	}
	if err != nil {
		return ShortenResult{}, err
	}

	_ = f.auditor.Notify(ctx, audit.NewEvent(audit.ActionShorten, userID, rawURL))
	return ShortenResult{ShortURL: shortURL}, nil
}

// ExpandURL возвращает исходный URL по идентификатору короткой ссылки.
func (f *ShortenerFacade) ExpandURL(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", service.ErrInvalidInput
	}

	originalURL, err := f.shortener.GetOriginal(ctx, id)
	if err != nil {
		return "", err
	}

	userID, _ := auth.UserIDFromContext(ctx)
	_ = f.auditor.Notify(ctx, audit.NewEvent(audit.ActionFollow, userID, originalURL))
	return originalURL, nil
}

// ListUserURLs возвращает URL аутентифицированного пользователя.
func (f *ShortenerFacade) ListUserURLs(ctx context.Context) ([]service.UserURLOutput, error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, auth.ErrNoUserID
	}
	return f.shortener.GetUserURLs(ctx, userID)
}
