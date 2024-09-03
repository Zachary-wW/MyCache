package mycache

import pb "mycache/mycachepb"

// 根据传入的key选择对应节点的PeerGetter方法
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	// Get(group string, key string) ([]byte, error)
	Get(in *pb.Request, out *pb.Response) error
}
