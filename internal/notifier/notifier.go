package notifier

import (
	"context"

	"go-cloud/internal/queue"
)

type Notifier interface {
	Send(ctx context.Context, message queue.NotificationMessage) error
}
