package mycache

// 负责与外部交互，控制缓存存储和获取的主流程

import (
	"fmt"
	"log"
	pb "mycache/mycachepb"
	"mycache/singleflight"
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
	getter Getter     // 本地数据源获取方法
	mcache mainCache  // 并发LRU-K
	peers  PeerPicker // 远程节点资源获取
	loader *singleflight.Group
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
		loader: &singleflight.Group{},
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

// 注入接口
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("Register PeerPicker More Than Once")
	}

	g.peers = peers
}

func (g *Group) Load(key string) (value ByteView, err error) {
	// 使用singleflight 针对多个请求相同的key 无论是远程读取还是本地获取都只执行一次
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			// 首先选取哪一个远程节点
			if peer, ok := g.peers.PickPeer(key); ok {
				// 从远程节点获取cache
				if value, err = g.GetFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[MyCache] Failed to get from peer", err)
			}
		}

		return g.GetLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}

	return
}

// 从远程节点中获取cache
func (g *Group) GetFromPeer(peer PeerGetter, key string) (ByteView, error) {
	// protobuf的request
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}

	res := &pb.Response{}
	err := peer.Get(req, res)
	// bytes, err := peer.Get(g.name, key)

	if err != nil {
		return ByteView{}, err
	}

	return ByteView{bytes: res.Value}, nil
}

// 本地获取节点 例如本地数据库
func (g *Group) GetLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)

	if err != nil {
		return ByteView{}, err
	}

	cv := ByteView{bytes: cloneBytes(bytes)}
	g.populateCache(key, cv) // 缓存
	log.Println("[MyCache] Get locally and populate!")

	return cv, nil
}

// 添加到本地cache中
func (g *Group) populateCache(key string, bytes ByteView) {
	g.mcache.Add(key, bytes)
}
