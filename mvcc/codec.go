package mvcc

import "encoding/binary"

// 这一文件集中存放 mvcc 内部使用的二进制编解码工具：
//   - KeyValue 的整体序列化 / 反序列化（存到 value bucket）
//   - 一个 key 对应的 revision 列表的编解码（存到 key bucket）
//   - int64 → big-endian 8 字节的小工具 u64（bucket key 的统一编码方式）
// 这些函数不依赖 Store / bbolt 事务，纯粹是字节布局，放在独立文件里方便阅读和测试。

// encodeKV 把 KeyValue 序列化为 [keyLen|key|valueLen|value|createRev|modRev]。
// 字节序统一使用 big-endian，便于 bbolt cursor 时按 revision 升序遍历。
func encodeKV(kv KeyValue) []byte {
	buf := make([]byte, 0, 32+len(kv.Key)+len(kv.Value))
	var tmp [8]byte
	binary.BigEndian.PutUint32(tmp[:4], uint32(len(kv.Key)))
	buf = append(buf, tmp[:4]...)
	buf = append(buf, kv.Key...)
	binary.BigEndian.PutUint32(tmp[:4], uint32(len(kv.Value)))
	buf = append(buf, tmp[:4]...)
	buf = append(buf, kv.Value...)
	binary.BigEndian.PutUint64(tmp[:], uint64(kv.CreateRevision))
	buf = append(buf, tmp[:]...)
	binary.BigEndian.PutUint64(tmp[:], uint64(kv.ModRevision))
	buf = append(buf, tmp[:]...)
	return buf
}

// decodeKV 是 encodeKV 的反向操作。
func decodeKV(b []byte) KeyValue {
	var kv KeyValue
	kl := binary.BigEndian.Uint32(b[0:4])
	b = b[4:]
	kv.Key = append([]byte(nil), b[:kl]...)
	b = b[kl:]
	vl := binary.BigEndian.Uint32(b[0:4])
	b = b[4:]
	kv.Value = append([]byte(nil), b[:vl]...)
	b = b[vl:]
	kv.CreateRevision = int64(binary.BigEndian.Uint64(b[0:8]))
	b = b[8:]
	kv.ModRevision = int64(binary.BigEndian.Uint64(b[0:8]))
	return kv
}

// encodeRevs 把一个 key 历史上所有写入的 revision 列表打包成连续的 8 字节块。
// 顺序与追加顺序一致（即按时间升序），列表末尾就是最新的 rev。
func encodeRevs(revs []int64) []byte {
	buf := make([]byte, 8*len(revs))
	for i, r := range revs {
		binary.BigEndian.PutUint64(buf[i*8:], uint64(r))
	}
	return buf
}

func decodeRevs(b []byte) []int64 {
	n := len(b) / 8
	out := make([]int64, n)
	for i := 0; i < n; i++ {
		out[i] = int64(binary.BigEndian.Uint64(b[i*8:]))
	}
	return out
}

// u64 把 int64 编成 big-endian 的 8 字节切片，主要用于：
//   - value bucket 的 key（rev → KV）
//   - meta bucket 的 current_rev 值
func u64(v int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}
