package loki

import (
	"sync"
)

type locker struct {
	mutex        sync.Mutex
	mutexesByKey map[string]*sync.Mutex
}

func newLocker() *locker {
	return &locker{mutexesByKey: make(map[string]*sync.Mutex)}
}

func (l *locker) obtain(key string) *sync.Mutex {
	l.mutex.Lock()
	mutex, ok := l.mutexesByKey[key]
	if !ok {
		mutex = &sync.Mutex{}
		l.mutexesByKey[key] = mutex
	}
	l.mutex.Unlock()

	return mutex
}

func (l *locker) release(key string) {
	l.mutex.Lock()
	delete(l.mutexesByKey, key)
	l.mutex.Unlock()
}
