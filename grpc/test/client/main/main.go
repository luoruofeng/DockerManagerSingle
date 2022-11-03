package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	pb "github.com/luoruofeng/dockermanagersingle/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	serverAddr  = flag.String("addr", "localhost:8997", "The server address in the format of host:port")
	containerId = "eee86007800a"
)

func Operation(client pb.DockerHandleClient) {
	//i will send cmds
	datas := []*pb.OperationRequest_Data{
		{[]byte("pwd \n cd usr\n")},
		{[]byte("rm test\n")},
		{[]byte("touch luoruofeng\n")},
		{[]byte("ls\n")},
		{[]byte("echo testecho\n")},
		{[]byte("Exit\n")},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 65*time.Second)
	defer cancel()
	stream, err := client.Operation(ctx)
	if err != nil {
		log.Fatalf("call operation failed: %v", err)
	}
	waitc := make(chan struct{})

	// get info from grpc server
	go func() {
		defer close(waitc)
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				fmt.Println("ohhhhh,i get EOF")
				return
			}
			if err != nil {
				log.Printf("stream recv data failed: %v\n", err.Error())
				return
			}
			fmt.Printf("Got data: %v\nGot meta: %v\n\n", in.GetData(), in.GetMeta())

			//quit
			if in.GetMeta() != nil && in.GetMeta().Code == -1 {
				fmt.Println("I got meta and i'm quit. error message is :" + in.GetMeta().GetErrormes())
				return
			}
		}
	}()

	//send container id
	cidreq := &pb.OperationRequest{
		Info: &pb.OperationRequest_ContainerId{ContainerId: containerId},
	}
	if err := stream.Send(cidreq); err != nil {
		log.Fatalf("client send container id failed: stream.Send(%v) failed: %v\n", containerId, err)
	}

	//send cmds
	for _, cmd := range datas {
		orequest := &pb.OperationRequest{
			Info: cmd,
		}
		if err := stream.Send(orequest); err != nil {
			log.Fatalf("send cmd failed: stream.Send(%v) failed: %v", cmd, err)
		}
		time.Sleep(time.Second * 1)
	}

	time.Sleep(time.Second * 8)
	fmt.Println("closesend!!!!!!")
	stream.CloseSend()
	<-waitc
	fmt.Println("Done")
}

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := pb.NewDockerHandleClient(conn)
	// op
	Operation(client)

}
