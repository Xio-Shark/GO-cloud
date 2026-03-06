package redisrepo

import (
	"context"
	"fmt"
	"time"

	"go-cloud/internal/repository"

	goredis "github.com/redis/go-redis/v9"
)

type LockRepository struct {
	client *goredis.Client
}

func NewLockRepository(client *goredis.Client) repository.LockRepository {
	return &LockRepository{client: client}
}

func (r *LockRepository) AcquireTaskDispatchLock(ctx context.Context, taskID int64, ttl time.Duration) (bool, error) {
	return r.client.SetNX(ctx, fmt.Sprintf("lock:scheduler:task:%d", taskID), "1", ttl).Result()
}
