package singleflight

import "sync"

// 我们并发了 N 个请求 ?key=Tom，8003 节点向 8001 同时发起了 N 次请求。
// 假设对数据库的访问没有做任何限制的，很可能向数据库也发起 N 次请求，
// 容易导致缓存击穿和穿透。即使对数据库做了防护，HTTP 请求是非常耗费资源的操作，
// 针对相同的 key，8003 节点向 8001 发起三次请求也是没有必要的。
// 那这种情况下，我们如何做到只向远端节点发起一次请求呢？

// 如果现在 我们使用singleflight有多个请求同时访问相同的key
// 每个请求都调用Do方法，第一个获取到锁的请求进行初始化map和添加一个goroutine

// 正在进行中或者已经结束的请求
type call struct {
	wg  sync.WaitGroup // 避免重入
	val interface{}
	err error
}

// singleflight 的主数据结构
type Group struct {
	mu sync.Mutex       // 保护m的并发读写安全
	m  map[string]*call // 不同的key对应不同的call
}

// 针对相同的key 无论Do被调用多少次 fn只会调用一次
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()

	if g.m == nil {
		g.m = make(map[string]*call) // 延迟初始化
	}

	// 其他routine
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()         // 请求进行中
		return c.val, c.err // 结束就返回
	}

	// 首先获取到lock的
	c := new(call)
	c.wg.Add(1)  // 添加一个goroutine
	g.m[key] = c // 表明key正在处理
	g.mu.Unlock()

	c.val, c.err = fn() //调用fn 发起请求
	c.wg.Done()         // 请求结束 计数器-1

	g.mu.Lock()
	delete(g.m, key) // 以免占用内存 同时可以保证key的最新性
	g.mu.Unlock()

	return c.val, c.err
}
