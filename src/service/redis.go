package service

import (
	"context"
	"time"

	"github.com/bsm/redislock"
	"github.com/fxamacker/cbor/v2"
	"github.com/redis/go-redis/v9"
	"github.com/teambition/gear"

	"github.com/yiwen-ai/yiwen-api/src/conf"
	"github.com/yiwen-ai/yiwen-api/src/util"
)

func init() {
	util.DigProvide(NewRedis)
	util.DigProvide(NewLocker)
}

type Redis struct {
	prefix string
	cli    *redis.Client
}

func NewRedis() *Redis {
	cfg := conf.Config.Redis
	client := redis.NewClient(&redis.Options{
		Network:  "tcp",
		Addr:     cfg.Node,
		Protocol: 3,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		panic(err)
	}

	return &Redis{
		prefix: cfg.Prefix,
		cli:    client,
	}
}

func (s *Redis) GetCBOR(ctx context.Context, key string, val any) error {
	data, err := s.cli.Get(ctx, s.prefix+key).Bytes()
	if err == redis.Nil {
		return gear.ErrNotFound.WithMsgf("key %q not found", key)
	} else if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	if err = cbor.Unmarshal(data, val); err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return nil
}

func (s *Redis) SetCBOR(ctx context.Context, key string, val any, ttl uint) error {
	data, err := cbor.Marshal(val)
	if err != nil {
		return gear.ErrBadRequest.From(err)
	}
	err = s.cli.Set(ctx, s.prefix+key, data, time.Duration(ttl)*time.Second).Err()
	if err != nil {
		return gear.ErrInternalServerError.From(err)
	}
	return nil
}

type Locker struct {
	prefix string
	locker *redislock.Client
}

func NewLocker() *Locker {
	cfg := conf.Config.Redis
	client := redis.NewClient(&redis.Options{
		Network:  "tcp",
		Addr:     cfg.Node,
		Protocol: 3,
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
