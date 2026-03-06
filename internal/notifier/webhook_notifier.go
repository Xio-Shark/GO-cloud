package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"go-cloud/internal/queue"
)

type WebhookNotifier struct {
	client *http.Client
}

func NewWebhookNotifier(timeout time.Duration) *WebhookNotifier {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &WebhookNotifier{
		client: &http.Client{Timeout: timeout},
	}
}

func (n *WebhookNotifier) Send(ctx context.Context, message queue.NotificationMessage) error {
	if message.CallbackURL == "" {
		return errors.New("callback_url is required")
	}
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, message.CallbackURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := n.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		return errors.New(response.Status)
	}
	return nil
}
