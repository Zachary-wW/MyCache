package lru

import "container/list"

// 缓存类型的接口，抽象形式
type cacheValue interface {
	Len() int
}

type Cache struct {
	maxBytes     int64                              // 缓存的最大容量
	usedBytes    int64                              // 当前使用字节
	doublyll     *list.List                         // 双向list
	cacheMap     map[string]*list.Element           // hash
	onEvicted    func(key string, value cacheValue) // 回调函数
	historyCache HistoryCache                       // 历史队列，访问次数达到K次才能加入到Cache中
}

type HistoryCache struct {
	k         int
	maxBytes  int64
	usedBytes int64
	doublyll  *list.List // 历史队列的淘汰策略是FIFO
	cacheMap  map[string]*list.Element
	cnt       map[string]int // 对每个节点访问次数的记录
}

// 双向list里面元素结构体
// 为什么要存个key？按道理来说通过使用key访问map之后得到list里面的value不就行了？
// 当我们删除一个node的时候，如何找到对应node的key从而删除map元素？
// 只能再存放一个key到list中
type entry struct {
	key   string
	value cacheValue
}

// 初始化Cache
func New(maxBytes int64, onEnvicted func(key string, value cacheValue), k int) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		doublyll:  list.New(),
		cacheMap:  make(map[string]*list.Element),
		onEvicted: onEnvicted, // 回调函数的作用是在缓存条目被移除时对其进行操作，例如日志记录、释放资源等

		historyCache: HistoryCache{
			k:        k,
			maxBytes: maxBytes,
			doublyll: list.New(),
			cacheMap: make(map[string]*list.Element),
			cnt:      make(map[string]int),
		},
	}
}

// 查找key对应的cache值
func (c *Cache) Get(key string) (value cacheValue, ok bool) {
	if _, ok = c.cacheMap[key]; ok {
		// 如果缓存命中
		listEle := c.cacheMap[key]
		c.doublyll.MoveToFront(listEle)
		kv := listEle.Value.(*entry)
		return kv.value, ok
	} else {
		// 缓存未命中 去查看历史队列是否存在，访问到k次加入缓存中
		if _, ok = c.historyCache.cacheMap[key]; ok {
			c.historyCache.cnt[key]++
			listEle := c.historyCache.cacheMap[key]
			kv := listEle.Value.(*entry)

			if c.historyCache.cnt[key] >= c.historyCache.k {
				c.AddToCache(kv.key, kv.value)
				// 从历史节点删除
				c.historyCache.doublyll.Remove(listEle)
				c.historyCache.usedBytes -= int64(len(kv.key)) + int64(kv.value.Len())
				delete(c.historyCache.cacheMap, kv.key)
				delete(c.historyCache.cnt, kv.key)
			} else {
				// 如果没有达到K次，把元素放在末尾，最晚被FIFO淘汰
				c.historyCache.doublyll.MoveToBack(listEle)
			}
			return kv.value, ok
		} else {
			return
		}
	}
	// return
}

func (c *Cache) Add(key string, value cacheValue) {
	if _, ok := c.cacheMap[key]; ok {
		// 缓存命中，移到前面，更新value
		listEle := c.cacheMap[key]
		c.doublyll.MoveToFront(listEle)
		kv := listEle.Value.(*entry)
		c.usedBytes += (int64(value.Len()) - int64(kv.value.Len()))
		kv.value = value
	} else {
		// 缓存未命中 加入到历史队列中（lru是直接加入到cache中）
		// listEle := c.doublyll.PushFront(&entry{key, value})
		// c.cacheMap[key] = listEle
		// c.usedBytes += (int64(len(key)) + int64(value.Len()))
		if _, ok = c.historyCache.cacheMap[key]; !ok {
			// 没有在历史队列中找到 新增
			listEle := c.historyCache.doublyll.PushBack(&entry{key, value})
			c.historyCache.cacheMap[key] = listEle
			c.historyCache.cnt[key]++
			c.historyCache.usedBytes += int64(len(key)) + int64(value.Len())
			// 判断内存是否被用完
			if c.historyCache.maxBytes != 0 && c.historyCache.usedBytes > c.historyCache.maxBytes {
				c.RemoveHistoryCacheOldest()
			}
		} else {
			// 在历史队列中找到，更新value 增加次数 放至队尾
			c.historyCache.cnt[key]++
			listEle := c.historyCache.cacheMap[key]
			c.historyCache.doublyll.MoveToBack(listEle)
			kv := listEle.Value.(*entry)
			kv.value = value
			c.historyCache.usedBytes += int64(value.Len()) - int64(kv.value.Len())
		}

		// 判断是否能够加入cache中
		if c.historyCache.cnt[key] >= c.historyCache.k {
			c.AddToCache(key, value)
			listEle := c.historyCache.cacheMap[key]
			kv := listEle.Value.(*entry)
			c.historyCache.doublyll.Remove(listEle)
			c.historyCache.usedBytes -= int64(len(kv.key)) + int64(kv.value.Len())
			delete(c.historyCache.cacheMap, kv.key)
			delete(c.historyCache.cnt, kv.key)
		}
	}
}

func (c *Cache) AddToCache(key string, value cacheValue) {
	listEle := c.doublyll.PushFront(&entry{key, value})
	c.cacheMap[key] = listEle
	c.usedBytes += int64(len(key)) + int64(value.Len())

	// maxBytes == 0 意味着不限制内存大小
	for c.maxBytes != 0 && c.usedBytes > c.maxBytes {
		c.RemoveCacheOldest()
	}
}

func (c *Cache) RemoveCacheOldest() {
	listEle := c.doublyll.Back()
	if listEle != nil {
		c.doublyll.Remove(listEle)
		kv := listEle.Value.(*entry)
		delete(c.cacheMap, kv.key) // 使用listEle的entry的key删除map里面的value
		c.usedBytes -= (int64(len(kv.key)) + int64(kv.value.Len()))

		if c.onEvicted != nil {
			c.onEvicted(kv.key, kv.value) // 回调函数不为空，就执行
		}
	}
}

func (c *Cache) RemoveHistoryCacheOldest() {
	listEle := c.historyCache.doublyll.Front() // FIFO的淘汰策略
	if listEle != nil {
		kv := listEle.Value.(*entry)
		c.historyCache.doublyll.Remove(listEle)
		c.historyCache.usedBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		delete(c.historyCache.cacheMap, kv.key)
		delete(c.historyCache.cnt, kv.key)
		if c.onEvicted != nil {
			c.onEvicted(kv.key, kv.value)
		}
	}
}

func (c *Cache) GetCacheLen() int {
	return c.doublyll.Len()
}
