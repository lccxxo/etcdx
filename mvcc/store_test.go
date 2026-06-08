package mvcc

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close(); os.RemoveAll(dir) })
	return s
}

func TestPutGet(t *testing.T) {
	s := newTestStore(t)
	rev, prev, err := s.Put([]byte("foo"), []byte("bar"))
	if err != nil {
		t.Fatal(err)
	}
	if rev != 1 {
		t.Fatalf("want rev=1, got %d", rev)
	}
	if prev != nil {
		t.Fatalf("first put should have no prev, got %+v", prev)
	}

	kv, err := s.Get([]byte("foo"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(kv.Value, []byte("bar")) {
		t.Fatalf("value mismatch: %s", kv.Value)
	}
	if kv.CreateRevision != 1 || kv.ModRevision != 1 {
		t.Fatal("rev fields wrong")
	}
}

func TestOverwriteReturnsPrev(t *testing.T) {
	s := newTestStore(t)
	s.Put([]byte("k"), []byte("v1"))
	rev, prev, err := s.Put([]byte("k"), []byte("v2"))
	if err != nil {
		t.Fatal(err)
	}
	if rev != 2 {
		t.Fatalf("want rev=2, got %d", rev)
	}
	if prev == nil || !bytes.Equal(prev.Value, []byte("v1")) {
		t.Fatalf("prev should be v1, got %+v", prev)
	}
	kv, _ := s.Get([]byte("k"))
	if !bytes.Equal(kv.Value, []byte("v2")) {
		t.Fatal("latest value should be v2")
	}
	if kv.CreateRevision != 1 || kv.ModRevision != 2 {
		t.Fatal("create/mod rev wrong")
	}
}

func TestGetMissing(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get([]byte("nope"))
	if err != ErrKeyNotFound {
		t.Fatalf("want ErrKeyNotFound, got %v", err)
	}
}
