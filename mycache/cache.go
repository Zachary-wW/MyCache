package mycache

/*
cache.go 中 封装了缓存的Add和Get方法以及添加了互斥锁
外界不需要考虑并发的问题
*/

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
		// 延迟初始化，减少程序内存开销
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
