package lock

import (
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
	pid = os.Getpid()
	seq uint64
)

// redisLock redis锁
type redisLock struct {
	client *redis.Client
}

// NewRedisLock 初始化redis锁
func NewRedisLock(client *redis.Client) Locker {
	return &redisLock{client: client}
}

func (l *redisLock) Lock(key string, ttl time.Duration) (func() error, bool, error) {
	reqID := fmt.Sprintf("%d-%d-%d", time.Now().UnixNano(), pid, atomic.AddUint64(&seq, 1))
	success, err := l.client.SetNX(key, reqID, ttl).Result()
	if err != nil {
		return nil, false, err
	}
	if !success {
		return nil, false, nil
	}
	return func() error {
		return l.client.Eval(redisUnlockScript, []string{key}, reqID).Err()
	}, true, nil
}
