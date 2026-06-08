package mvcc

import (
	"encoding/binary"
	"errors"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketKey   = []byte("key")
	bucketValue = []byte("value")
	bucketMeta  = []byte("meta")

	keyCurrentRev = []byte("current_rev")
)

var ErrKeyNotFound = errors.New("key not found")

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

func (s *Store) Put(key, value []byte) (rev int64, prev *KeyValue, err error) {
	err = s.db.Update(func(tx *bolt.Tx) error {
		rev = nextRevision(tx)

		keyB := tx.Bucket(bucketKey)
		valB := tx.Bucket(bucketValue)

		oldRevs := decodeRevs(valB.Get(key))
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

func (s *Store) Get(key []byte) (*KeyValue, error) {
	var out *KeyValue
	err := s.db.View(func(tx *bolt.Tx) error {
		revs := decodeRevs(tx.Bucket(bucketKey).Get(key))
		if len(revs) == 0 {
			return ErrKeyNotFound
		}
		latest := revs[len(revs)-1]
		kv := decodeKV(tx.Bucket(bucketValue).Get(u64(latest)))
		out = &kv
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
