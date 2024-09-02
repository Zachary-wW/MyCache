package lru

import (
	"reflect"
	"testing"
)

type String string

func (d String) Len() int {
	return len(d)
}

// 只针对于LRU的测试,即LRU-1

func TestGet(t *testing.T) {
	lru := New(int64(0), nil, 1) // 0表示无限制
	lru.Add("key1", String("123"))
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "123" {
		t.Fatalf("cache hit key1=123 failed")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

func TestRemoveOldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := "value1", "value2", "value3"
	cap := len(k1 + v1 + k2 + v2)
	lru := New(int64(cap), nil, 1)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3))

	if _, ok := lru.Get("key1"); ok || lru.GetCacheLen() != 2 {
		t.Fatalf("RemoveOldest key1 failed")
	}
}

func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value cacheValue) {
		keys = append(keys, key)
	}
	lru := New(int64(10), callback, 1)
	lru.Add("key1", String("123456"))
	lru.Add("k2", String("k2"))
	lru.Add("k3", String("k3"))
	lru.Add("k4", String("k4"))

	except := []string{"key1", "k2"}

	if !reflect.DeepEqual(except, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", except)
	}

}
