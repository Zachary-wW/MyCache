package mycache

import (
	"fmt"
	"log"
	"testing"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db)) // 创建db长度的map
	myCache := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[mycache_test:][DB] Searching key", key)
			if v, ok := db[key]; ok { // find in db
				if _, ok := loadCounts[key]; !ok { // if not in the loadCount, add
					loadCounts[key] = 0
				}
				loadCounts[key] += 1 // add count
				return []byte(v), nil
			}
			return nil, fmt.Errorf("[mycache_test:][DB] %s not exist in DB", key)
		}))

	for k, v := range db {
		if view, err := myCache.Get(k); err != nil || view.String() != v {
			t.Fatal("[mycache_test:] Failed to get value")
		} // load data
		if _, err := myCache.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("[mycache_test:] Cache %s miss", k)
		} // hit cache
	}

	log.Println("After getting locally: ")

	for k, v := range db {
		log.Println("key is", k)
		if view, err := myCache.Get(k); err != nil || view.String() != v {
			t.Fatal("[mycache_test:] Failed to get value")
		} // load data
		// if _, err := myCache.Get(k); err != nil || loadCounts[k] > 1 {
		// 	t.Fatalf("[mycache_test:] Cache %s miss", k)
		// } // hit cache
	}

	if view, err := myCache.Get("unknown"); err == nil {
		t.Fatalf("[mycache_test:] The value of unknow should be empty, but %s got", view)
	}
}
