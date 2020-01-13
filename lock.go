package lock

import "time"

//Locker 锁接口
type Locker interface {
	Lock(key string, ttl time.Duration) (func() error, bool, error)
}
