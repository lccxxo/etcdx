package server

import (
	"context"

	pb "github.com/lccxxo/etcdx/api/v1"
)

type Server struct {
	pb.UnimplementedKVServer
}

func New() *Server {
	return &Server{}
}

func (s *Server) Put(ctx context.Context, r *pb.PutRequest) (*pb.PutResponse, error) {

	return nil, nil
}

func (s *Server) Range(ctx context.Context, r *pb.RangeRequest) (*pb.RangeResponse, error) {
	return nil, nil
}

func (s *Server) Delete(ctx context.Context, r *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	return nil, nil
}
