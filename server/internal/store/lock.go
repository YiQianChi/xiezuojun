package store

import (
	"sync"
)

// UserLocks 按 userID 分组的读写锁池
type UserLocks struct {
	locks sync.Map // string → *sync.RWMutex
}

// getLock 获取或创建指定用户的锁
func (ul *UserLocks) getLock(userID string) *sync.RWMutex {
	val, _ := ul.locks.LoadOrStore(userID, &sync.RWMutex{})
	return val.(*sync.RWMutex)
}

// RLock 获取用户读锁
func (ul *UserLocks) RLock(userID string) {
	ul.getLock(userID).RLock()
}

// RUnlock 释放用户读锁
func (ul *UserLocks) RUnlock(userID string) {
	ul.getLock(userID).RUnlock()
}

// Lock 获取用户写锁
func (ul *UserLocks) Lock(userID string) {
	ul.getLock(userID).Lock()
}

// Unlock 释放用户写锁
func (ul *UserLocks) Unlock(userID string) {
	ul.getLock(userID).Unlock()
}
