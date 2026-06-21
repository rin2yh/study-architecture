// Package broker はドメインイベントを運ぶ Redis 接続を組み立てる。
package broker

import (
	"errors"
	"os"

	"github.com/redis/go-redis/v9"
)

func NewClient() (*redis.Client, error) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		return nil, errors.New("REDIS_URL is required")
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	return redis.NewClient(opt), nil
}
