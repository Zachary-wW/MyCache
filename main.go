package main

import (
	"flag"
	"fmt"
	"log"
	"mycache"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

// 创建group名称为scores

func createGroup() *mycache.Group {
	// 使用匿名函数给Getter赋值
	return mycache.NewGroup("scores", 2<<10, mycache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[DB] Searching key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// addr是server端地址
func startCacheServer(addr string, addrs []string, gee *mycache.Group) {
	// 使用addr初始化server
	peers := mycache.NewHTTPPool(addr)
	// 将addrs作为远程节点
	peers.SetPeers(addrs...)
	// peers是httppool类型，里面实现了PickPeer功能
	gee.RegisterPeers(peers)

	log.Println("Mycache is running at", addr)

	// ListenAndServe listens on the TCP network address addr and
	// then calls [Serve] with handler to handle requests on incoming connections

	// [7:] means that localhost:8003
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(apiAddr string, g *mycache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := g.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}

func main() {
	var port int
	var api bool

	flag.IntVar(&port, "port", 8001, "Mycache Server Port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	if api {
		go startAPIServer(apiAddr, gee)
	}
	startCacheServer(addrMap[port], []string(addrs), gee)

}
