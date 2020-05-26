package lock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis"
)

var (
	redisUnlockScript = `
local prev = redis.call("get", KEYS[1]);
if (prev ~= false and prev == ARGV[1]) then
	return redis.call("del", KEYS[1]);
end
return 0
`
	redisRenewScript = `
local prev = redis.call("get", KEYS[1]);
if (prev ~= false and prev == ARGV[1]) then
	return redis.call("expire", KEYS[1], ARGV[2]);
end
return 0
`
	pid = os.Getpid()
	seq uint64
	// ErrConcurrentConflict ErrConcurrentConflict
	ErrConcurrentConflict = errors.New("ErrConcurrentConflict")
)

// redisLock redis锁
type redisLock struct {
	client *redis.Client
}

// NewRedisLock 初始化redis锁
func NewRedisLock(client *redis.Client) Locker {
	return &redisLock{client: client}
}

func (l *redisLock) Lock(key string, ttl time.Duration, onLost func()) (func() error, error) {
	reqID := fmt.Sprintf("%d-%d-%d", time.Now().UnixNano(), pid, atomic.AddUint64(&seq, 1))
	success, err := l.client.SetNX(key, reqID, ttl).Result()
	if err != nil {
		return nil, err
	}
	if !success {
		return nil, ErrConcurrentConflict
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		timer := time.NewTimer(ttl)
		defer timer.Stop()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		defer cancel()
		onLost = func() {
			if onLost != nil {
				onLost()
			}
		}

		for {
			ok, err := l.client.Eval(redisRenewScript, []string{key}, reqID, 60).Int()
			if err != nil {
				select {
				case <-timer.C:
					onLost()
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
				continue
			}
			if ok == 0 {
				onLost()
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				onLost()
				return
			case <-ticker.C:
			}
		}
	}()
	return func() error {
		cancel()
		return l.client.Eval(redisUnlockScript, []string{key}, reqID).Err()
	}, nil
}

func (l *redisLock) DoInLock(rawCtx context.Context, key string, ttl time.Duration, handler func(ctx context.Context) error) error {
	ctx, cancel := context.WithCancel(rawCtx)
	defer cancel()
	unlock, err := l.Lock(key, ttl, cancel)
	if err != nil {
		return err
	}
	defer unlock()
	if err := handler(ctx); err != nil {
		return err
	}
	return nil
}
