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

	kvs, _, err := s.Range([]byte("foo"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(kvs) != 1 {
		t.Fatalf("want 1 kv, got %d", len(kvs))
	}
	kv := kvs[0]
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
	kvs, _, _ := s.Range([]byte("k"), nil)
	if len(kvs) != 1 {
		t.Fatalf("want 1 kv, got %d", len(kvs))
	}
	kv := kvs[0]
	if !bytes.Equal(kv.Value, []byte("v2")) {
		t.Fatal("latest value should be v2")
	}
	if kv.CreateRevision != 1 || kv.ModRevision != 2 {
		t.Fatal("create/mod rev wrong")
	}
}

// ---------------------------------------------------------------------------
// Range
// ---------------------------------------------------------------------------

func TestRangeSingleKey(t *testing.T) {
	s := newTestStore(t)
	s.Put([]byte("foo"), []byte("bar"))

	kvs, rev, err := s.Range([]byte("foo"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if rev != 1 {
		t.Fatalf("want rev=1, got %d", rev)
	}
	if len(kvs) != 1 || !bytes.Equal(kvs[0].Value, []byte("bar")) {
		t.Fatalf("range single key mismatch: %+v", kvs)
	}
}

func TestRangeSingleKeyMissing(t *testing.T) {
	s := newTestStore(t)
	s.Put([]byte("foo"), []byte("bar"))

	kvs, rev, err := s.Range([]byte("nope"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if rev != 1 {
		t.Fatalf("rev should still report current store rev, got %d", rev)
	}
	if len(kvs) != 0 {
		t.Fatalf("want empty result, got %+v", kvs)
	}
}

func TestRangePrefix(t *testing.T) {
	s := newTestStore(t)
	s.Put([]byte("a1"), []byte("1"))
	s.Put([]byte("a2"), []byte("2"))
	s.Put([]byte("b1"), []byte("3"))

	kvs, _, err := s.Range([]byte("a"), []byte("b"))
	if err != nil {
		t.Fatal(err)
	}
	if len(kvs) != 2 {
		t.Fatalf("want 2 hits, got %d: %+v", len(kvs), kvs)
	}
	if !bytes.Equal(kvs[0].Key, []byte("a1")) || !bytes.Equal(kvs[1].Key, []byte("a2")) {
		t.Fatalf("prefix order wrong: %+v", kvs)
	}
}

func TestRangeAll(t *testing.T) {
	s := newTestStore(t)
	s.Put([]byte("a"), []byte("1"))
	s.Put([]byte("m"), []byte("2"))
	s.Put([]byte("z"), []byte("3"))

	kvs, _, err := s.Range([]byte{0}, []byte{0})
	if err != nil {
		t.Fatal(err)
	}
	if len(kvs) != 3 {
		t.Fatalf("want 3, got %d: %+v", len(kvs), kvs)
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDeleteSingleKey(t *testing.T) {
	s := newTestStore(t)
	s.Put([]byte("foo"), []byte("bar"))

	deleted, _, rev, err := s.Delete([]byte("foo"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("want deleted=1, got %d", deleted)
	}
	if rev != 2 {
		t.Fatalf("want rev=2 (1 put + 1 delete), got %d", rev)
	}
	if kvs, _, _ := s.Range([]byte("foo"), nil); len(kvs) != 0 {
		t.Fatalf("foo should be gone, got %+v", kvs)
	}
}

func TestDeleteRange(t *testing.T) {
	s := newTestStore(t)
	s.Put([]byte("a1"), []byte("1"))
	s.Put([]byte("a2"), []byte("2"))
	s.Put([]byte("b1"), []byte("3"))

	deleted, _, _, err := s.Delete([]byte("a"), []byte("b"))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 2 {
		t.Fatalf("want deleted=2, got %d", deleted)
	}
	if kvs, _, _ := s.Range([]byte("a1"), nil); len(kvs) != 0 {
		t.Fatal("a1 should be gone")
	}
	if kvs, _, _ := s.Range([]byte("a2"), nil); len(kvs) != 0 {
		t.Fatal("a2 should be gone")
	}
	kvs, _, err := s.Range([]byte("b1"), nil)
	if err != nil || len(kvs) != 1 || !bytes.Equal(kvs[0].Value, []byte("3")) {
		t.Fatalf("b1 should survive, got kvs=%+v err=%v", kvs, err)
	}
}

func TestDeleteReturnsPrev(t *testing.T) {
	s := newTestStore(t)
	s.Put([]byte("k"), []byte("v"))

	_, prevs, _, err := s.Delete([]byte("k"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(prevs) != 1 {
		t.Fatalf("want 1 prev, got %d", len(prevs))
	}
	if !bytes.Equal(prevs[0].Value, []byte("v")) {
		t.Fatalf("prev value mismatch: %s", prevs[0].Value)
	}
	if prevs[0].ModRevision != 1 {
		t.Fatalf("prev ModRevision should be the original put rev, got %d", prevs[0].ModRevision)
	}
}

func TestDeleteMissingNoError(t *testing.T) {
	s := newTestStore(t)
	deleted, prevs, _, err := s.Delete([]byte("nope"), nil)
	if err != nil {
		t.Fatalf("delete on missing key must not error: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("want deleted=0, got %d", deleted)
	}
	if len(prevs) != 0 {
		t.Fatalf("want no prevs, got %+v", prevs)
	}
}
