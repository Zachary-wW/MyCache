package mycache

// 提供被其他节点访问的能力(基于http)

import (
	"fmt"
	"io"
	"log"
	"mycache/consistenthash"
	pb "mycache/mycachepb"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
)

const (
	defaultBasePath = "/_mycache/"
	defaultReplicas = 50
)

type HTTPPool struct {
	self        string // 自己的地址 IP+port
	basePath    string
	mu          sync.Mutex                     // 假设有多个client向你发送请求
	chash       *consistenthash.ConsistentHash // 选择对应的节点
	httpGetters map[string]*httpGetter         // 远程节点和Get方法映射
}

// httpGetter实际上就是对应远程节点的http client
type httpGetter struct {
	baseURL string
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// 日志信息
func (hp *HTTPPool) Log(format string, v ...any) {
	log.Printf("[Server %s] %s", hp.self, fmt.Sprintf(format, v...))
}

// 实现 http.Handler 接口
func (hp *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 确保访问的是缓存对应的path
	if !strings.HasPrefix(r.URL.Path, hp.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	// 打印方法和路径
	hp.Log("(In ServeHTTP) %s %s", r.Method, r.URL.Path)
	// <basePath>/<Group>/<key>
	parts := strings.SplitN(r.URL.Path[len(hp.basePath):], "/", 2)
	// 不匹配上述形式
	if len(parts) != 2 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	// 如果匹配
	groupName := parts[0]
	key := parts[1]
	// 找到对应group的缓存
	group := GetGroup(groupName)
	// 找不到组
	if group == nil {
		http.Error(w, "No such Group: "+groupName, http.StatusNotFound)
		return
	}
	// 找到组
	cv, err := group.Get(key)
	// 如果返回error 说明内部错误
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 使用protobuf包装
	body, err := proto.Marshal(&pb.Response{Value: cv.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 一切没问题
	// application/octet-stream表示所有其他情况的默认值
	// 一种未知的文件类型应当使用此类型
	w.Header().Set("Content-Tye", "application/octet-stream")
	// w.Write(cv.ByteSlice())
	w.Write(body)
}

// 实例化一致性hash 添加节点 为每个节点创建一个httpGetter（client）
func (hp *HTTPPool) SetPeers(peers ...string) {
	hp.mu.Lock()
	defer hp.mu.Unlock()

	hp.chash = consistenthash.New(defaultReplicas, nil) // 采用默认的hash函数
	hp.chash.Add(peers...)                              // 添加节点
	hp.httpGetters = make(map[string]*httpGetter)       // 延迟初始化

	for _, peer := range peers {
		hp.httpGetters[peer] = &httpGetter{baseURL: peer + hp.basePath}
	}
}

// 对于传入的key找到真实节点 返回PeerGetter接口
func (hp *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	hp.mu.Lock()
	defer hp.mu.Unlock()

	// 从哈希环中寻找节点
	if peer := hp.chash.Get(key); peer != "" && peer != hp.self {
		hp.Log("Pick Peer %s", peer)
		// 返回对应节点的httpgetter 即 client
		return hp.httpGetters[peer], true
	}

	return nil, false
}

// 这行代码是一个类型断言，它将*HTTPPool指针断言为PeerPicker接口类型。这通常用于接口的实现声明，
// 这里它表明HTTPPool实现了PeerPicker接口。
var _ PeerPicker = (*HTTPPool)(nil)

// ----------------------http client---------------------------

// 实现PeerGetter接口
// func (hg *httpGetter) Get(group string, key string) ([]byte, error) {
func (hg *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	info := fmt.Sprintf(
		"%v%v/%v", // %v按原本值输出
		hg.baseURL,
		url.QueryEscape(in.GetGroup()), // QueryEscape 会对字符串进行转义处理，以便将其安全地放入 URL 查询中
		url.QueryEscape(in.GetKey()),
	)
	// Get方法
	res, err := http.Get(info)
	// 有错误
	if err != nil {
		return err
	}

	defer res.Body.Close()
	// 不是200
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("[ERROR] Server returned: %v", res.Status)
	}
	// ok了 读取数据
	bytes, err := io.ReadAll(res.Body) // read until an error or EOF and returns the data it read
	if err != nil {
		return fmt.Errorf("[ERROR] Reading response body: %v", err)
	}
	// Decode
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

var _ PeerGetter = (*httpGetter)(nil)

// var _ PeerGetter = (*httpGetter)(nil)
// var _ PeerPicker = (*HTTPPool)(nil)
