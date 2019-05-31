package lockotron

import (
	"github.com/sasha-s/go-deadlock"
)

type locker struct {
	mutex        deadlock.Mutex
	mutexesByKey map[string]*deadlock.Mutex
}

func newLocker() *locker {
	return &locker{mutexesByKey: make(map[string]*deadlock.Mutex)}
}

func (l *locker) obtain(key string) *deadlock.Mutex {
	l.mutex.Lock()
	mutex, ok := l.mutexesByKey[key]
	if !ok {
		mutex = &deadlock.Mutex{}
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
