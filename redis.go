package lock

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisLockScript = `
local prev = redis.call("get", KEYS[1]);
if (prev == false) then
	return redis.call("set", KEYS[1], ARGV[1], "ex", ARGV[2]);
end
if (prev == ARGV[1]) then
	return redis.call("set", KEYS[1], ARGV[1], "ex", ARGV[2]);
end
return "FAIL"
`
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
)

// redisLock redis lock
type redisLock struct {
	client *redis.Client
}

// NewRedisLock 初始化redis锁
func NewRedisLock(client *redis.Client) Locker {
	return &redisLock{client: client}
}

func (l *redisLock) doLock(key, value string, expirationSeconds int, options Options) (bool, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), options.timeout)
	defer cancel()

	result, err := l.client.Eval(ctx, redisLockScript, []string{key}, value, expirationSeconds).Result()
	if err != nil {
		return false, err
	}
	resultInt := result.(string)
	return resultInt == "OK", nil

}

func (l *redisLock) doRenew(key, value string, expiration int, options Options) (bool, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), options.timeout)
	defer cancel()

	result, err := l.client.Eval(ctx, redisRenewScript, []string{key}, value, expiration).Result()
	if err != nil {
		return false, err
	}
	resultInt := result.(int64)
	return resultInt == 1, nil
}

func (l *redisLock) doUnlock(key, value string, options Options) error {
	ctx, cancel := context.WithTimeout(context.TODO(), options.timeout)
	defer cancel()
	if err := l.client.Eval(ctx, redisUnlockScript, []string{key}, value).Err(); err != nil {
		return err
	}
	return nil
}

func (l *redisLock) TryLock(key string, expirationSeconds int, onLost func(), opts ...Option) (bool, func() error, error) {
	if expirationSeconds <= 0 {
		expirationSeconds = defaultExpirationSeconds
	}

	options := defaultOptions()
	options.apply(opts...)

	expiration := time.Duration(expirationSeconds) * time.Second
	renewInterval := expiration / 4
	reqID := options.idGenerator()
	success, err := l.doLock(key, reqID, expirationSeconds, options)
	if err != nil {
		return false, nil, err
	}
	if !success {
		return false, nil, nil
	}

	ctxUnlock, unLockCtxCancelFunc := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(renewInterval)
		defer ticker.Stop()
		curOnLost := func() {
			if onLost != nil {
				onLost()
			}
		}

		for range ticker.C {
			select {
			case <-ctxUnlock.Done():
				return
			default:
				success, err := l.doRenew(key, reqID, expirationSeconds, options)
				if err != nil {
					continue
				}
				if !success {
					curOnLost()
					return
				}
			}
		}
	}()
	return true, func() error {
		unLockCtxCancelFunc()
		if err := l.doUnlock(key, reqID, options); err != nil {
			return err
		}
		return nil
	}, nil
}

func (l *redisLock) Lock(key string, expirationSeconds int, onLost func(), opts ...Option) (func() error, error) {
	options := defaultOptions()
	options.apply(opts...)
	for {
		success, unlock, err := l.TryLock(key, expirationSeconds, onLost, opts...)
		if err != nil {
			return nil, err
		}
		if success {
			return unlock, nil
		}
		time.Sleep(options.retryInterval)
	}
}

func (l *redisLock) LockContext(ctx context.Context, key string, expirationSeconds int, onLost func(), opts ...Option) (func() error, error) {
	options := defaultOptions()
	options.apply(opts...)
	for {
		success, unlock, err := l.TryLock(key, expirationSeconds, onLost, opts...)
		if err != nil {
			return nil, err
		}
		if success {
			return unlock, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		time.Sleep(options.retryInterval)
	}
}

func (l *redisLock) InLock(key string, expirationSeconds int, handler func(ctx context.Context) error, opts ...Option) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	unlock, err := l.Lock(key, expirationSeconds, cancel, opts...)
	if err != nil {
		return err
	}
	defer unlock()
	if err := handler(ctx); err != nil {
		return err
	}
	return nil
}

func (l *redisLock) InLockContext(ctx context.Context, key string, expirationSeconds int, handler func(ctx context.Context) error, opts ...Option) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	unlock, err := l.LockContext(ctx, key, expirationSeconds, cancel, opts...)
	if err != nil {
		return err
	}
	defer unlock()
	if err := handler(ctx); err != nil {
		return err
	}
	return nil
}

func (l *redisLock) TryInLock(key string, expirationSeconds int, handler func(ctx context.Context) error, opts ...Option) (bool, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	success, unlock, err := l.TryLock(key, expirationSeconds, cancel, opts...)
	if err != nil {
		return false, err
	}
	if !success {
		return false, nil
	}
	defer unlock()
	if err := handler(ctx); err != nil {
		return true, err
	}
	return true, nil
}

func (l *redisLock) TryInLockContext(ctx context.Context, key string, expirationSeconds int, handler func(ctx context.Context) error, opts ...Option) (bool, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	success, unlock, err := l.TryLock(key, expirationSeconds, cancel, opts...)
	if err != nil {
		return false, err
	}
	if !success {
		return false, nil
	}
	defer unlock()
	if err := handler(ctx); err != nil {
		return true, err
	}
	return true, nil
}
