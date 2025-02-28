package lock

import (
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestRedisLock(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	lock := NewRedisLock(client)
	success, unlock, err := lock.TryLock("test", 10, nil)

	if err != nil {
		t.Fatalf("lock expected err:nil, got:%#v", err)
	}
	if !success {
		t.Fatal("lock expected success:true, got:false")
	}
	defer unlock()
}
