package lock

import (
	"testing"
	"time"

	"github.com/go-redis/redis"
)

func TestRedisLock(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	lock := NewRedisLock(client)
	unlock, err := lock.Lock("test", time.Second*3, nil)

	if err != nil {
		t.Fatalf("lock expected err:nil, got:%#v", err)
	}

	_, err = lock.Lock("test", time.Second*3, nil)

	if err != ErrConcurrentConflict {
		t.Fatal("lock expected success:false, got:true")
	}
	time.Sleep(time.Second * 3)
	unlock()
	unlock, err = lock.Lock("test", time.Second*3, nil)

	if err != nil {
		t.Fatalf("lock expected err:nil, got:%#v", err)
	}

	unlock()
}
