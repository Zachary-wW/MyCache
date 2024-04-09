package lru

import "container/list"

// 缓存类型的接口，抽象形式
type cacheValue interface {
	Len() int
}

type Cache struct {
	maxBytes  int64                              // 缓存的最大容量
	usedBytes int64                              // 当前使用字节
	doublyll  *list.List                         // 双向list
	cache     map[string]*list.Element           // hash
	onEvicted func(key string, value cacheValue) // 回调函数
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
func New(maxBytes int64, onEnvicted func(key string, value cacheValue)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		doublyll:  list.New(),
		cache:     make(map[string]*list.Element),
		onEvicted: onEnvicted,
	}
}

// 查找key对应的cache值
func (c *Cache) Get(key string) (value cacheValue, ok bool) {
	if listEle, ok := c.cache[key]; ok {
		c.doublyll.MoveToFront(listEle)
		kv := listEle.Value.(*entry)
		return kv.value, ok
	}
	return
}

func (c *Cache) RemoveOldest() {
	listEle := c.doublyll.Back()
	if listEle != nil {
		c.doublyll.Remove(listEle)
		kv := listEle.Value.(*entry)
		delete(c.cache, kv.key) // 使用listEle的entry的key删除map里面的value
		c.usedBytes -= (int64(len(kv.key)) + int64(kv.value.Len()))

		if c.onEvicted != nil {
			c.onEvicted(kv.key, kv.value) // 回调函数不为空，就执行
		}
	}
}

func (c *Cache) Add(key string, value cacheValue) {
	if listEle, ok := c.cache[key]; ok {
		c.doublyll.MoveToFront(listEle)
		kv := listEle.Value.(*entry)
		c.usedBytes += (int64(value.Len()) - int64(kv.value.Len()))
		kv.value = value
	} else {
		listEle := c.doublyll.PushFront(&entry{key, value})
		c.cache[key] = listEle
		c.usedBytes += (int64(len(key)) + int64(value.Len()))
	}
	for c.maxBytes != 0 && c.maxBytes < c.usedBytes {
		c.RemoveOldest()
	}
}

func (c *Cache) GetCacheLen() int {
	return c.doublyll.Len()
}
