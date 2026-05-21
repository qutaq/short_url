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

	"github.com/hashicorp/go-retryablehttp"
)

const (
	// ActionShorten идентифицирует события аудита после сокращения URL.
	ActionShorten = "shorten"
	// ActionFollow идентифицирует события аудита после переходов по коротким ссылкам.
	ActionFollow = "follow"
)

const (
	remoteObserverRequestTimeout = 5 * time.Second
	remoteObserverRetryMax       = 3
	remoteObserverRetryWaitMin   = 100 * time.Millisecond
	remoteObserverRetryWaitMax   = time.Second
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
	observers []notifierObserver
}

type notifierObserver struct {
	observer Observer
	sem      chan struct{}
}

// NewNotifier создаёт Notifier для переданных наблюдателей.
func NewNotifier(observers ...Observer) *Notifier {
	notifier := &Notifier{
		observers: make([]notifierObserver, 0, len(observers)),
	}
	for _, observer := range observers {
		notifier.observers = append(notifier.observers, notifierObserver{
			observer: observer,
			sem:      make(chan struct{}, 1),
		})
	}
	return notifier
}

// Notify отправляет событие всем настроенным наблюдателям параллельно и объединяет ошибки.
func (n *Notifier) Notify(ctx context.Context, event Event) error {
	if len(n.observers) == 0 {
		return nil
	}

	errCh := make(chan error, len(n.observers))
	var wg sync.WaitGroup

	for _, observer := range n.observers {
		wg.Add(1)
		go func(observer notifierObserver) {
			defer wg.Done()
			select {
			case observer.sem <- struct{}{}:
				defer func() { <-observer.sem }()
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}

			errCh <- observer.observer.Notify(ctx, event)
		}(observer)
	}

	wg.Wait()
	close(errCh)
	var err error
	for observerErr := range errCh {
		err = errors.Join(err, observerErr)
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
		url:    rawURL,
		client: newRetryableHTTPClient(),
	}, nil
}

func newRetryableHTTPClient() *http.Client {
	client := retryablehttp.NewClient()
	client.RetryMax = remoteObserverRetryMax
	client.RetryWaitMin = remoteObserverRetryWaitMin
	client.RetryWaitMax = remoteObserverRetryWaitMax
	client.Logger = nil
	client.HTTPClient = &http.Client{
		Timeout: remoteObserverRequestTimeout,
	}

	return client.StandardClient()
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
