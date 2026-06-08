package main

import (
	"flag"
	"log"
	"net"

	pb "github.com/lccxxo/etcdx/api/v1"
	"github.com/lccxxo/etcdx/server"
	"google.golang.org/grpc"
)

func main() {
	listen := flag.String("listen", ":2379", "grpc listen address")
	flag.Parse()

	lis, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	pb.RegisterKVServer(s, server.New())

	log.Printf("etcdx server listening on %s", *listen)
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
