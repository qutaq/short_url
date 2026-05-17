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
	ActionShorten = "shorten"
	ActionFollow  = "follow"
)

type Event struct {
	Timestamp int64  `json:"ts"`
	Action    string `json:"action"`
	UserID    string `json:"user_id,omitempty"`
	URL       string `json:"url"`
}

func NewEvent(action, userID, url string) Event {
	return Event{
		Timestamp: time.Now().Unix(),
		Action:    action,
		UserID:    userID,
		URL:       url,
	}
}

type Observer interface {
	Notify(ctx context.Context, event Event) error
}

type Notifier struct {
	observers []Observer
}

func NewNotifier(observers ...Observer) *Notifier {
	return &Notifier{observers: observers}
}

func (n *Notifier) Notify(ctx context.Context, event Event) error {
	if n == nil {
		return nil
	}

	var err error
	for _, observer := range n.observers {
		err = errors.Join(err, observer.Notify(ctx, event))
	}
	return err
}

type FileObserver struct {
	mu   sync.Mutex
	file *os.File
}

func NewFileObserver(path string) (*FileObserver, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open audit file: %w", err)
	}
	return &FileObserver{file: file}, nil
}

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

func (o *FileObserver) Close() error {
	if o == nil {
		return nil
	}
	return o.file.Close()
}

type RemoteObserver struct {
	url    string
	client *http.Client
}

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
