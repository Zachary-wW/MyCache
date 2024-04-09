package mycache

import (
	"fmt"
	"log"
	"sync"
)

// 首先定义一个接口 接口中具有一个Get方法
type Getter interface {
	Get(key string) ([]byte, error)
}

// 实现Getter中的Get方法
// 这是定义了一个函数类型，该类型为接口函数，因为实现了接口
// 如果有个函数test和GetterFunc参数返回值都相同，可以强制转换，之后可以作为Getter的参数传入

// 既能够将普通的函数类型（需类型转换）作为参数，
// 也可以将结构体作为参数，使用更为灵活，可读性也更好，这就是接口型函数的价值。
type GetterFunc func(key string) ([]byte, error)

// 回调函数
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

type Group struct {
	name   string
	getter Getter
	mcache mainCache
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}

	mu.Lock()
	defer mu.Unlock()

	g := &Group{
		name:   name,
		getter: getter,
		mcache: mainCache{cacheBytes: cacheBytes},
	}

	groups[name] = g

	return g
}

func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is empty")
	}

	if cv, ok := g.mcache.Get(key); ok {
		log.Println("[MyCache:] Hit Cache!")
		return cv, nil
	}

	return g.Load(key)
}

func (g *Group) Load(key string) (value ByteView, err error) {
	return g.GetLocally(key)
}

func (g *Group) GetLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key) // 调用接口Get方法

	if err != nil {
		return ByteView{}, err
	}

	cv := ByteView{bytes: cloneBytes(bytes)}
	g.populateCache(key, cv)
	log.Println("[MyCache:] Get locally and populate!")

	return cv, nil
}

func (g *Group) populateCache(key string, bytes ByteView) {
	g.mcache.Add(key, bytes)
}
