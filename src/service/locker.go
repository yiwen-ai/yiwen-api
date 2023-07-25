package service

import (
	"context"
	"time"

	"github.com/bsm/redislock"
	"github.com/redis/go-redis/v9"

	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

func init() {
	util.DigProvide(NewLocker)
}

type Locker struct {
	prefix string
	locker *redislock.Client
}

func NewLocker() *Locker {
	cfg := conf.Config.Redis
	client := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    cfg.Node,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		panic(err)
	}

	// Create a new lock client.
	locker := redislock.New(client)

	return &Locker{
		prefix: cfg.Prefix,
		locker: locker,
	}
}

func (s *Locker) Lock(ctx context.Context, key string, du time.Duration) (*redislock.Lock, error) {
	return s.locker.Obtain(ctx, s.prefix+key, du, nil)
}
