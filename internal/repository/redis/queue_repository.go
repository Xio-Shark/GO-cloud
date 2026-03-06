package redisrepo

import (
	"context"
	"time"

	"go-cloud/internal/repository"

	goredis "github.com/redis/go-redis/v9"
)

const (
	taskQueueKey         = "queue:task:execution"
	notificationQueueKey = "queue:task:notification"
)

type QueueRepository struct {
	client *goredis.Client
}

func NewQueueRepository(client *goredis.Client) repository.QueueRepository {
	return &QueueRepository{client: client}
}

func (r *QueueRepository) EnqueueTask(ctx context.Context, payload []byte) error {
	return r.client.LPush(ctx, taskQueueKey, payload).Err()
}

func (r *QueueRepository) DequeueTask(ctx context.Context, timeout time.Duration) ([]byte, error) {
	return r.pop(ctx, taskQueueKey, timeout)
}

func (r *QueueRepository) EnqueueNotification(ctx context.Context, payload []byte) error {
	return r.client.LPush(ctx, notificationQueueKey, payload).Err()
}

func (r *QueueRepository) DequeueNotification(ctx context.Context, timeout time.Duration) ([]byte, error) {
	return r.pop(ctx, notificationQueueKey, timeout)
}

func (r *QueueRepository) pop(ctx context.Context, key string, timeout time.Duration) ([]byte, error) {
	result, err := r.client.BRPop(ctx, timeout, key).Result()
	if err == goredis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(result) != 2 {
		return nil, nil
	}
	return []byte(result[1]), nil
}
