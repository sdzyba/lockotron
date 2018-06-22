package lockotron

import (
	"time"
)

type item struct {
	value interface{}
	ttl   int64
}

func newItem(value interface{}, ttl time.Duration) *item {
	return &item{value: value, ttl: time.Now().Add(ttl).UnixNano()}
}

func (i *item) isExpirable() bool {
	if i.ttl == int64(NoTTL) {
		return false
	}

	return true
}
