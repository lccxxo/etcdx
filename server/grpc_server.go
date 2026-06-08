package server

import (
	"context"

	pb "github.com/lccxxo/etcdx/api/v1"
	"github.com/lccxxo/etcdx/mvcc"
)

type Server struct {
	pb.UnimplementedKVServer
	store *mvcc.Store
}

func New(store *mvcc.Store) *Server {
	return &Server{store: store}
}

func (s *Server) Put(ctx context.Context, req *pb.PutRequest) (*pb.PutResponse, error) {
	rev, prev, err := s.store.Put(req.Key, req.Value)
	if err != nil {
		return nil, err
	}
	resp := &pb.PutResponse{
		Revision: rev,
	}
	if req.PrevKv && prev != nil {
		resp.PrevKv = toPBKV(prev)
	}
	return resp, nil
}

func (s *Server) Range(ctx context.Context, req *pb.RangeRequest) (*pb.RangeResponse, error) {
	kvs, rev, err := s.store.Range(req.Key, req.RangeEnd)
	if err != nil {
		return nil, err
	}
	return &pb.RangeResponse{
		Kvs:      toPBKVs(kvs),
		Revision: rev,
	}, nil
}

func (s *Server) Delete(ctx context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	deleted, prevs, rev, err := s.store.Delete(req.Key, req.RangeEnd)
	if err != nil {
		return nil, err
	}
	resp := &pb.DeleteResponse{
		Deleted:  deleted,
		Revision: rev,
	}
	if req.PrevKv {
		resp.PrevKvs = toPBKVs(prevs)
	}
	return resp, nil
}

func toPBKV(kv *mvcc.KeyValue) *pb.KeyValue {
	if kv == nil {
		return nil
	}
	return &pb.KeyValue{
		Key:            kv.Key,
		Value:          kv.Value,
		CreateRevision: kv.CreateRevision,
		ModRevision:    kv.ModRevision,
	}
}

func toPBKVs(kvs []mvcc.KeyValue) []*pb.KeyValue {
	out := make([]*pb.KeyValue, 0, len(kvs))
	for i := range kvs {
		out = append(out, toPBKV(&kvs[i]))
	}
	return out
}
