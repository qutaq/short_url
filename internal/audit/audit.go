package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

const (
	// ActionShorten идентифицирует события аудита после сокращения URL.
	ActionShorten = "shorten"
	// ActionFollow идентифицирует события аудита после переходов по коротким ссылкам.
	ActionFollow = "follow"
)

// Event описывает одно аудируемое действие пользователя.
type Event struct {
	// Timestamp содержит Unix-время создания события.
	Timestamp int64 `json:"ts"`
	// Action содержит тип события.
	Action string `json:"action"`
	// UserID идентифицирует пользователя, связанного с событием.
	UserID string `json:"user_id,omitempty"`
	// URL содержит URL, которого касается событие.
	URL string `json:"url"`
}

// NewEvent создаёт Event с текущим Unix-временем.
func NewEvent(action, userID, url string) Event {
	return Event{
		Timestamp: time.Now().Unix(),
		Action:    action,
		UserID:    userID,
		URL:       url,
	}
}

// Observer получает события аудита.
type Observer interface {
	// Notify обрабатывает одно событие аудита.
	Notify(ctx context.Context, event Event) error
}

// Notifier передаёт события аудита одному или нескольким наблюдателям.
type Notifier struct {
	observers []Observer
}

// NewNotifier создаёт Notifier для переданных наблюдателей.
func NewNotifier(observers ...Observer) *Notifier {
	return &Notifier{observers: observers}
}

// Notify отправляет событие каждому настроенному наблюдателю и объединяет ошибки.
func (n *Notifier) Notify(ctx context.Context, event Event) error {
	var err error
	for _, observer := range n.observers {
		err = errors.Join(err, observer.Notify(ctx, event))
	}
	return err
}

// FileObserver записывает события аудита в файл в формате JSON Lines.
type FileObserver struct {
	mu   sync.Mutex
	file *os.File
}

// NewFileObserver открывает файл по указанному пути для добавления событий аудита.
func NewFileObserver(path string) (*FileObserver, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open audit file: %w", err)
	}
	return &FileObserver{file: file}, nil
}

// Notify записывает событие в файл аудита.
func (o *FileObserver) Notify(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}
	data = append(data, '\n')

	o.mu.Lock()
	defer o.mu.Unlock()
	if _, err := o.file.Write(data); err != nil {
		return fmt.Errorf("write audit event: %w", err)
	}
	return nil
}

// Close закрывает нижележащий файл аудита.
func (o *FileObserver) Close() error {
	if o == nil {
		return nil
	}
	return o.file.Close()
}

// RemoteObserver отправляет события аудита на удалённую HTTP-точку приёма.
type RemoteObserver struct {
	url    string
	client *http.Client
}

// NewRemoteObserver создаёт RemoteObserver для указанного URL.
func NewRemoteObserver(rawURL string) (*RemoteObserver, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid audit url: %q", rawURL)
	}

	return &RemoteObserver{
		url: rawURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}, nil
}

// Notify отправляет событие на удалённую точку приёма аудита в формате JSON.
func (o *RemoteObserver) Notify(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create audit request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return fmt.Errorf("send audit event: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("send audit event: unexpected status %d", resp.StatusCode)
	}
	return nil
}
