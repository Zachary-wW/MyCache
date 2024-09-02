package mycache

import (
	"mycache/lru"
	"sync"
)

type mainCache struct {
	mu         sync.Mutex
	lru        *lru.Cache
	cacheBytes int64
}

func (mc *mainCache) Add(key string, value ByteView) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.lru == nil {
		mc.lru = lru.New(mc.cacheBytes, nil, 1)
	}
	mc.lru.Add(key, value)
}

func (mc *mainCache) Get(key string) (value ByteView, ok bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.lru == nil {
		return
	}

	if cv, ok := mc.lru.Get(key); ok {
		return cv.(ByteView), ok
	}
	return
}
