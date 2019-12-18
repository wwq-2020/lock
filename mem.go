package lock

import "time"

func init() {
	Register("mem", newMemLock)
}

type memLock struct {
	tokenCh chan struct{}
}

// newMemLock 初始化内存锁
func newMemLock() Locker {
	return &memLock{
		tokenCh: make(chan struct{}, 1),
	}
}

// Lock 锁
func (l *memLock) Lock() {
	l.tokenCh <- struct{}{}
}

// Lock 锁
func (l *memLock) LockWithTimeout(duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case l.tokenCh <- struct{}{}:
		return true
	case <-timer.C:
		return false
	}
}

// TryLock 尝试锁
func (l *memLock) TryLock() bool {
	select {
	case l.tokenCh <- struct{}{}:
		return true
	default:
		return false
	}
}

// Unlock 解锁
func (l *memLock) Unlock() {
	select {
	case <-l.tokenCh:
	default:
		panic("unlock of unlocked lock")
	}
}
