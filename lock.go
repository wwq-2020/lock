package lock

import (
	"fmt"
	"time"
)

//Locker 锁接口
type Locker interface {
	Lock()
	LockWithTimeout(duration time.Duration) bool
	TryLock() bool
	Unlock()
}

type lockerInitFun func() Locker

var lockerInitFunMap = make(map[string]lockerInitFun)

// Register 注册锁初始化方法
func Register(typ string, lockerInitFun lockerInitFun) {
	_, exist := lockerInitFunMap[typ]
	if exist {
		panic(fmt.Sprintf("dup for %s", typ))
	}
	lockerInitFunMap[typ] = lockerInitFun
}

// New 初始化锁
func New(typ string) Locker {
	lockerInitFun, exist := lockerInitFunMap[typ]
	if !exist {
		lockerInitFun = newMemLock
	}
	return lockerInitFun()
}
