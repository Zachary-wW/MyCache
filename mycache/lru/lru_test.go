package lru

import (
	"reflect"
	"testing"
)

type String string // 为了构造一个匹配接口的函数

func (d String) Len() int {
	return len(d)
} // get the length of String

func TestGet(t *testing.T) {
	lru := New(int64(0), nil) // maxBytes 设置为0，代表不对内存大小设限
	lru.Add("key1", String("1234"))
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("Cache hit key1=1234 failed!")
	} // what is the meaning of string(v.(String))
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("Cache miss key2 failed")
	}
}

func TestRemoveOldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "key3"
	v1, v2, v3 := "value1", "value2", "value3"
	capa := len(k1 + k2 + v1 + v2)
	lru := New(int64(capa), nil)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3))

	if _, ok := lru.Get("key1"); ok || lru.GetCacheLen() != 2 {
		t.Fatalf("Remove Function Failed!")
	}
}

func TestOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value cacheValue) {
		keys = append(keys, key)
	}
	lru := New(int64(20), callback)
	lru.Add("key1", String("nihao"))
	lru.Add("key2", String("nihuai"))
	lru.Add("key3", String("nichou"))
	lru.Add("key4", String("nicai"))

	expect := []string{"key1", "key2"} // first two pop

	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equal to %s", expect)
	}
}
