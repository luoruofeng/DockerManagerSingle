package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pb "github.com/luoruofeng/dockermanagersingle/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var (
	serverAddr  = flag.String("addr", "localhost:8997", "The server address in the format of host:port")
	containerId = "testreids"
)

func Operation(client pb.DockerHandleClient) {

	ctx, cancel := context.WithTimeout(context.Background(), 650*time.Second)
	defer cancel()
	stream, err := client.Operation(ctx)
	if err != nil {
		fmt.Printf("call operation failed: %v\n", err)
		return
	}
	waitc := make(chan struct{})

	//i will send cmds
	// datas := []*pb.OperationRequest_Data{
	// 	{[]byte("pwd \n cd usr\n")},
	// 	{[]byte("rm test\n")},
	// 	{[]byte("touch luoruofeng\n")},
	// 	{[]byte("ls\n")},
	// 	{[]byte("echo testecho\n")},
	// 	// {[]byte("Exit\n")},
	// }

	//throw stdin send cmd
	bufReader := bufio.NewReader(os.Stdin)
	go func() {
		for {
			bs := make([]byte, 1024)
			n, err := bufReader.Read(bs)
			if err != nil {
				fmt.Println(err)
				waitc <- struct{}{}
				return
			} else {
				ord := pb.OperationRequest_Data{Data: bs[:n]}
				orequest := &pb.OperationRequest{
					Info: &ord,
				}
				if err := stream.Send(orequest); err != nil {
					fmt.Printf("send cmd failed: stream.Send(%v) failed: %v\n", string(bs[:n]), err)
					waitc <- struct{}{}
					return
				}
			}
		}
	}()

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
				fmt.Printf("stream recv data failed: %v\n", err.Error())
				return
			}
			// fmt.Printf("Got data: %v\nGot meta: %v\n\n", in.GetData(), in.GetMeta())

			if in.GetMeta() != nil {
				fmt.Printf("meta:%v\n", in.GetMeta())
			} else {
				fmt.Print(in.GetData())
			}

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
		fmt.Printf("client send container id failed: stream.Send(%v) failed: %v\n", containerId, err)
		return
	}

	//send cmds
	// for _, cmd := range datas {
	// 	time.Sleep(time.Second * 1)
	// 	orequest := &pb.OperationRequest{
	// 		Info: cmd,
	// 	}
	// 	if err := stream.Send(orequest); err != nil {
	// 		fmt.Fatalf("send cmd failed: stream.Send(%v) failed: %v", cmd, err)
	// 	}
	// }
	// time.Sleep(time.Second * 4)
	<-waitc
	fmt.Println("closesend!!!!!!")
	stream.CloseSend()
	fmt.Println("Done")
}

func ImagePull(client pb.DockerHandleClient) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	in := pb.PullImageWithLogRequest{
		ImageName:    "redis",
		ImageVersion: "6.0",
	}
	fmt.Printf("start pull image. tag:%v:%v\n", in.ImageName, in.ImageVersion)
	stream, err := client.PullImageWithLog(ctx, &in)
	if err != nil {
		fmt.Printf("call operation failed: %v\n", err)
		return
	}

	for {
		reply, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				fmt.Println("EOF")
				return
			} else {
				fmt.Println(err)
				return
			}
		}
		if reply.GetMeta() != nil {
			fmt.Printf("meta:%v\n", reply.GetMeta())
		} else {
			fmt.Println(reply.GetData())
		}
	}

}

// 没有使用的一般拦截器
func logInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	fmt.Println("开始设置一般拦截器")
	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	end := time.Now()
	fmt.Printf("RPC: %s, 开始: %s, 结束: %s, 错误: %v", method, start.Format("Basic"), end.Format(time.RFC3339), err)
	return err
}

// 设置日志流拦截器
// wrappedStream  wraps around the embedded grpc.ClientStream, and intercepts the RecvMsg and
// SendMsg method call.
type wrappedStream struct {
	grpc.ClientStream
}

func newWrappedStream(s grpc.ClientStream) grpc.ClientStream {
	return &wrappedStream{s}
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	fmt.Printf("获取到信息啦：(Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	return w.ClientStream.RecvMsg(m)
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	fmt.Printf("发送出信息啦：(Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	return w.ClientStream.SendMsg(m)
}

func streamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	fmt.Println("开始设置流拦截器")

	fmt.Println("检查服务器监控状态")
	healthClient := grpc_health_v1.NewHealthClient(cc)
	response, err := healthClient.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		log.Printf("%v", err)
	}
	log.Printf("目前服务器状态：%v", response)

	s, err := streamer(ctx, desc, cc, method, opts...)
	if err != nil {
		return nil, err
	}
	fmt.Println("结束设置流拦截器")
	return newWrappedStream(s), nil
}

// 超时时间
const defaultTestTimeout = 4 * time.Second

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// 设置interceptors
	opts = append(opts, grpc.WithChainStreamInterceptor(streamInterceptor))

	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		fmt.Printf("fail to dial: %v", err)
		return
	}
	defer conn.Close()

	client := pb.NewDockerHandleClient(conn)
	// op
	Operation(client)
	// pull image
	// ImagePull(client)

}
