package lock

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// Locker 锁接口
type Locker interface {
	Lock(key string, expirationSeconds int, onLost func(), opts ...Option) (func() error, error)

	LockContext(ctx context.Context, key string, expirationSeconds int, onLost func(), opts ...Option) (func() error, error)

	TryLock(key string, expirationSeconds int, onLost func(), opts ...Option) (bool, func() error, error)

	InLock(key string, expirationSeconds int, handler func(ctx context.Context) error, opts ...Option) error

	InLockContext(ctx context.Context, key string, expirationSeconds int, handler func(ctx context.Context) error, opts ...Option) error

	TryInLock(key string, expirationSeconds int, handler func(ctx context.Context) error, opts ...Option) (bool, error)

	TryInLockContext(ctx context.Context, key string, expirationSeconds int, handler func(ctx context.Context) error, opts ...Option) (bool, error)
}

const (
	defaultExpirationSeconds = 10
	defaultTimeout           = 3 * time.Second
	defaultRetryInterval     = time.Millisecond * 100
)

var (
	defaultIDGenerator = func() string {
		return fmt.Sprintf("%d-%d-%d", time.Now().Unix(), pid, atomic.AddUint64(&seq, 1))
	}
	pid = os.Getpid()
	seq uint64
)

type Options struct {
	timeout       time.Duration
	retryInterval time.Duration
	idGenerator   func() string
}

func (o *Options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

type Option func(*Options)

func WithRetryInterval(retryInterval time.Duration) Option {
	return func(o *Options) {
		if retryInterval > 0 {
			o.retryInterval = retryInterval
		}
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		if timeout > 0 {
			o.timeout = timeout
		}
	}
}

func WithIDGenerator(idGenerator func() string) Option {
	return func(o *Options) {
		if idGenerator != nil {
			o.idGenerator = idGenerator
		}
	}
}

func defaultOptions() Options {
	return Options{
		timeout:       defaultTimeout,
		retryInterval: defaultRetryInterval,
		idGenerator:   defaultIDGenerator,
	}
}
