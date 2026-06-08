package main

import (
	"flag"
	"log"
	"net"
	"os"
	"path/filepath"

	pb "github.com/lccxxo/etcdx/api/v1"
	"github.com/lccxxo/etcdx/mvcc"
	"github.com/lccxxo/etcdx/server"
	"google.golang.org/grpc"
)

func main() {
	listen := flag.String("listen", ":2379", "grpc listen address")
	flag.Parse()

	dataDir := flag.String("data-dir", "./data", "")
	flag.Parse()
	os.MkdirAll(*dataDir, 0755)
	store, err := mvcc.Open(filepath.Join(*dataDir, "kv.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	lis, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	pb.RegisterKVServer(s, server.New(store))

	log.Printf("etcdx server listening on %s", *listen)
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
