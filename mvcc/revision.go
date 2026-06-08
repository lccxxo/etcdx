package mvcc

import (
	"encoding/binary"

	bolt "go.etcd.io/bbolt"
)

func nextRevision(tx *bolt.Tx) int64 {
	meta := tx.Bucket(bucketMeta)
	cur := int64(0)
	if v := meta.Get(keyCurrentRev); v != nil {
		cur = int64(binary.BigEndian.Uint64(v))
	}
	cur++
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(cur))
	_ = meta.Put(keyCurrentRev, buf)
	return cur
}

func u64(v int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	return buf
}
