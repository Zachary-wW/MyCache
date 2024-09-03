// 使用一致性hash解决只使用普通hash映射的缓存雪崩的问题
// （缓存雪崩是指缓存在同一时间内全部失效，造成瞬时对于DB的请求骤增，容易造成宕机）
// 一致性hash将节点常用的标识，例如节点名称、IP等计算hash值，映射到2^32的环中
// 同样的，计算key的hash值，进行相同的映射，这样一来，从key出发顺时针寻找的第一个节点就是需要访问的节点

// 此时，出现一个问题：数据倾斜的问题
// 数据倾斜就是说：因为映射的空间可以想象成一个环，那么如果节点的hash值都在右上角聚集，这就会导致分布在左下角的key每次
// 访问的都是右上角第一个节点，造成负载不均衡
// Solution: 使用虚拟节点，然后将真实节点和虚拟节点之间使用一个map进行维护

package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 注意是uint
// 计算hash值
type Hash func(data []byte) uint32

type ConsistentHash struct {
	hashfn      Hash
	replicas    int
	ring        []int // 为了之后排序
	dummyToreal map[int]string
}

// 初始化一致性哈希
func New(replicas int, fn Hash) *ConsistentHash {
	ch := &ConsistentHash{
		replicas:    replicas,
		hashfn:      fn,
		dummyToreal: make(map[int]string),
	}
	// 赋默认hash函数
	if ch.hashfn == nil {
		ch.hashfn = crc32.ChecksumIEEE
	}

	return ch
}

// 传入多个/一个 real node，然后创建replicas个dummy nodes
func (ch *ConsistentHash) Add(nodes ...string) {
	for _, node := range nodes {
		for i := 0; i < ch.replicas; i++ {
			hash := int(ch.hashfn([]byte(strconv.Itoa(i) + node))) // 将基数为10的数转换成字符串形式
			ch.ring = append(ch.ring, hash)
			ch.dummyToreal[hash] = node
		}
	}
	sort.Ints(ch.ring)
}

func (ch *ConsistentHash) Get(key string) string {
	// 环是空的
	if len(ch.ring) == 0 {
		return ""
	}
	// 不是空的 先获取key的hash值
	hash := int(ch.hashfn([]byte(key)))
	// 使用binary search 因为ring有序
	// 找到第一个大于key的node
	idx := sort.Search(len(ch.ring), func(i int) bool {
		return ch.ring[i] >= hash
	})
	// idx只是一个索引 realToDummy的key是hash值，也就是ring上的值，需要进行转换
	return ch.dummyToreal[ch.ring[idx%len(ch.ring)]]
}
