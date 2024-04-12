package mycache

// 封装一个字节数组用来表示缓存
type ByteView struct {
	bytes []byte
}

// 匹配接口
// 使用值传递 不改变缓存值
func (bv ByteView) Len() int {
	return len(bv.bytes)
}

// 克隆byte
func (bv ByteView) ByteSlice() []byte {
	return cloneBytes(bv.bytes)
}

// return the string type of data
func (bv ByteView) String() string {
	return string(bv.bytes)
}

func cloneBytes(bytes []byte) []byte {
	tmp := make([]byte, len(bytes))
	copy(tmp, bytes)
	return tmp
}
