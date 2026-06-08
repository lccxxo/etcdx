package mvcc

import (
	"encoding/binary"

	bolt "go.etcd.io/bbolt"
)

// revision 是 mvcc 层的"全局逻辑时钟"：每次 Put / Delete 都会让它 +1，
// 客户端拿这个数做事件去重、Watch 续传、CAS 等。
// 这里只关心计数语义；二进制编解码用 codec.go 里的 u64。

func nextRevision(tx *bolt.Tx) int64 {
	cur := currentRevision(tx) + 1
	_ = tx.Bucket(bucketMeta).Put(keyCurrentRev, u64(cur))
	return cur
}

func currentRevision(tx *bolt.Tx) int64 {
	v := tx.Bucket(bucketMeta).Get(keyCurrentRev)
	if v == nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(v))
}
