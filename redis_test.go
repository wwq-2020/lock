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
	unlock, success, err := lock.Lock("test", time.Second*3, nil)

	if err != nil {
		t.Fatalf("lock expected err:nil, got:%#v", err)
	}
	if !success {
		t.Fatal("lock expected success:true, got:false")
	}
	_, success, err = lock.Lock("test", time.Second*3, nil)

	if err != nil {
		t.Fatalf("lock expected err:nil, got:%#v", err)
	}
	if success {
		t.Fatal("lock expected success:false, got:true")
	}
	time.Sleep(time.Second * 3)
	unlock()
	unlock, success, err = lock.Lock("test", time.Second*3, nil)

	if err != nil {
		t.Fatalf("lock expected err:nil, got:%#v", err)
	}
	if !success {
		t.Fatal("lock expected success:true, got:false")
	}
	unlock()
}
