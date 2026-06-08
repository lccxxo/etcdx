package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	pb "github.com/lccxxo/etcdx/api/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	endpoint := flag.String("endpoint", "localhost:2379", "")
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	conn, err := grpc.Dial(*endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	cli := pb.NewKVClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch args[0] {
	case "put":
		if len(args) != 3 {
			usage()
		}
		resp, err := cli.Put(ctx, &pb.PutRequest{Key: []byte(args[1]), Value: []byte(args[2])})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("OK rev=", resp.Revision)
	case "get":
		if len(args) != 2 {
			usage()
		}
		resp, err := cli.Range(ctx, &pb.RangeRequest{Key: []byte(args[1])})
		if err != nil {
			log.Fatal(err)
		}
		for _, kv := range resp.Kvs {
			fmt.Printf("%s = %s\n", kv.Key, kv.Value)
		}
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: etcdxctl [--endpoint addr] <put|get> ...")
	os.Exit(2)
}
