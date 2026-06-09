package mvcc

import (
	"bytes"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketKey   = []byte("key")
	bucketValue = []byte("value")
	bucketMeta  = []byte("meta")

	keyCurrentRev = []byte("current_rev")
)

type KeyValue struct {
	Key            []byte
	Value          []byte
	CreateRevision int64
	ModRevision    int64
}

type Store struct {
	db *bolt.DB
}

func Open(path string) (*Store, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{bucketKey, bucketValue, bucketMeta} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Put(key, value []byte) (rev int64, prev *KeyValue, err error) {
	err = s.db.Update(func(tx *bolt.Tx) error {
		rev = nextRevision(tx)

		keyB := tx.Bucket(bucketKey)
		valB := tx.Bucket(bucketValue)

		oldRevs := decodeRevs(keyB.Get(key))
		createRev := rev
		if len(oldRevs) > 0 {
			old := decodeKV(valB.Get(u64(oldRevs[len(oldRevs)-1])))
			prev = &old
			createRev = old.CreateRevision
		}

		newKV := KeyValue{
			Key:            key,
			Value:          value,
			CreateRevision: createRev,
			ModRevision:    rev,
		}

		if err := valB.Put(u64(rev), encodeKV(newKV)); err != nil {
			return err
		}

		oldRevs = append(oldRevs, rev)
		return keyB.Put(key, encodeRevs(oldRevs))
	})

	return
}

// rangeKeys 在 key bucket 上按 [key, rangeEnd) 语义扫一遍，返回命中的所有 user key。
// 这是 Range 和 Delete 共用的扫描逻辑，抽出来避免两份重复实现。
//
// rangeEnd 的三种约定（与 .proto 注释一致）：
//   - len(rangeEnd) == 0           只命中 key 这一项（若存在）
//   - rangeEnd == [0x00]           无上界，命中所有 >= key 的 user key
//   - 其他                          半开区间 [key, rangeEnd)
//
// 注意：返回的是 key bucket 的 cursor key，bbolt 在事务外不保证有效，
// 因此这里 copy 一份独立切片返回，调用方可以在事务里继续 Get/Put/Delete 使用。
func rangeKeys(tx *bolt.Tx, key, rangeEnd []byte) [][]byte {
	keyB := tx.Bucket(bucketKey)

	if len(rangeEnd) == 0 {
		if keyB.Get(key) == nil {
			return nil
		}
		return [][]byte{append([]byte(nil), key...)}
	}

	unbounded := len(rangeEnd) == 1 && rangeEnd[0] == 0
	var out [][]byte
	c := keyB.Cursor()
	for k, _ := c.Seek(key); k != nil; k, _ = c.Next() {
		if !unbounded && bytes.Compare(k, rangeEnd) >= 0 {
			break
		}
		out = append(out, append([]byte(nil), k...))
	}
	return out
}

// Delete 在 [key, rangeEnd) 范围里硬删 key bucket 的条目，value bucket 的旧版本不动
// （后续 compaction 阶段统一清理）。语义与 Range 对称，rangeEnd 的三种约定一致。
//
// 不论命中数（哪怕 deleted=0）都会消耗一个 rev，这样调用方拿到的 rev 永远是单调递增的
// 全局逻辑时钟；这也和 etcd 行为对齐——一次 RPC 一个 rev。
//
// prevs 始终收集（即被删 KV 的最新版本快照），上层 grpc handler 再决定是否回传给客户端。
func (s *Store) Delete(key, rangeEnd []byte) (deleted int64, prevs []KeyValue, rev int64, err error) {
	err = s.db.Update(func(tx *bolt.Tx) error {
		rev = nextRevision(tx)
		keyB := tx.Bucket(bucketKey)
		valB := tx.Bucket(bucketValue)

		for _, k := range rangeKeys(tx, key, rangeEnd) {
			revs := decodeRevs(keyB.Get(k))
			if len(revs) == 0 {
				continue
			}
			prevs = append(prevs, decodeKV(valB.Get(u64(revs[len(revs)-1]))))
			if err := keyB.Delete(k); err != nil {
				return err
			}
			deleted++
		}
		return nil
	})
	return
}

func (s *Store) Range(key, rangeEnd []byte) (kvs []KeyValue, rev int64, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		rev = currentRevision(tx)
		keyB := tx.Bucket(bucketKey)
		valB := tx.Bucket(bucketValue)

		for _, k := range rangeKeys(tx, key, rangeEnd) {
			revs := decodeRevs(keyB.Get(k))
			if len(revs) == 0 {
				continue
			}
			kvs = append(kvs, decodeKV(valB.Get(u64(revs[len(revs)-1]))))
		}
		return nil
	})
	return
}
