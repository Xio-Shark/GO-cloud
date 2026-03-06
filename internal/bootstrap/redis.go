package bootstrap

import (
	goredis "github.com/redis/go-redis/v9"
)

func NewRedis(cfg Config) *goredis.Client {
	return goredis.NewClient(&goredis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
}
