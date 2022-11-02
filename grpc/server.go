package grpc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/luoruofeng/dockermanagersingle/container"

	pb "github.com/luoruofeng/dockermanagersingle/pb"
	"golang.org/x/sync/errgroup"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedDockerHandleServer
	ctx context.Context
	cm  container.ContainerManager
}

func createMetaDialogueReply() *pb.DialogueReply {
	return &pb.DialogueReply{
		Info: &pb.DialogueReply_Meta{
			Meta: &pb.DialogueReplyMeta{},
		},
	}
}

func createDataDialogueReply() *pb.DialogueReply {
	// var data *pb.DialogueReply_Data = new(pb.DialogueReply_Data)
	// var reply *pb.DialogueReply = &pb.DialogueReply{Info: data}
	// return reply
	var reply *pb.DialogueReply = &pb.DialogueReply{Info: &pb.DialogueReply_Data{}}
	return reply
}

func setMetaReply(m *pb.DialogueReply_Meta, errormes string, code int32, duration int32) {
	m.Meta.Code = code
	m.Meta.Errormes = errormes
	m.Meta.Duration = duration
}

func setDataReply(m *pb.DialogueReply_Data, data string) {
	m.Data = data
}

func getDuration(stime time.Time) int32 {
	return int32(time.Since(stime)/time.Millisecond) / 1000
}

func (s *server) Operation(stream pb.DockerHandle_OperationServer) error {
	log.Printf("start Operation handle. stream: %v\n", stream)

	// variable of meta for send
	replymeta := createMetaDialogueReply()
	m, _ := replymeta.Info.(*pb.DialogueReply_Meta)

	//get container id
	recvdata, err := stream.Recv()
	if err != nil {
		log.Println(err)
		setMetaReply(m, err.Error(), -1, 0)
		stream.Send(replymeta)
		return err
	}

	cid := recvdata.GetContainerId()
	log.Printf("get container id: %v \n", cid)
	if cid == "" {
		err := stream.Send(replymeta)
		if err != nil {
			log.Println(err.Error())
			setMetaReply(m, "send reply failed", -1, 0)
			stream.Send(replymeta)
			return err
		}
		setMetaReply(m, "container id is empty", -1, 0)
		stream.Send(replymeta)
		return errors.New("container id is empty")
	}

	stime := time.Now()
	//get container bash
	hijack, err := s.cm.BashContainer(cid)
	log.Printf("get container bash.\n")
	if err != nil {
		log.Println(err.Error())
		setMetaReply(m, "bash container failed. "+err.Error(), -1, getDuration(stime))
		stream.Send(replymeta)
		return err
	}
	defer hijack.Close()

	//get cmd line from grpc caller and send to hijack
	eg, ctx := errgroup.WithContext(context.Background())

	eg.Go(
		func() error {
			for {
				select {
				case <-ctx.Done():

					fmt.Println("---------------------1")
					return nil
				default:
					recvdata, err = stream.Recv()
					if err != nil {
						setMetaReply(m, "I got EOF and i'm DONE", -1, 0)
						stream.Send(replymeta)
						return err
					}
					rbs := recvdata.GetData()
					rs := string(rbs)
					if strings.ToLower(rs) == "exit" {
						fmt.Println("read EXIT from client")
						setMetaReply(m, "", 1, getDuration(stime))
						stream.Send(replymeta)
						return errors.New("user send exit")
					}
					_, err = hijack.Conn.Write(rbs)
					if err != nil {
						m, _ := replymeta.Info.(*pb.DialogueReply_Meta)
						dt := int32(time.Since(stime)/time.Millisecond) / 1000
						setMetaReply(m, "hijack write content failed. content:"+rs+" error:"+err.Error(), -1, dt)
						stream.Send(replymeta)
						return err
					}
				}
			}
		},
	)

	//read container data from hijact and write to grpc caller
	eg.Go(func() error {
		bufReader := bufio.NewReader(hijack.Reader)
		for {
			select {
			case <-ctx.Done():
				fmt.Println("---------------------")
				return nil
			default:
				line, _, err := bufReader.ReadLine()
				if err != nil {
					if err == io.EOF {
						setMetaReply(m, "", 1, getDuration(stime))
						stream.Send(replymeta)
						return err
					} else {
						setMetaReply(m, "hijack read content failed. content:"+string(line)+" error:"+err.Error(), -1, getDuration(stime))
						stream.Send(replymeta)
						return err
					}
				}

				datareply := createDataDialogueReply()
				d, ok := datareply.Info.(*pb.DialogueReply_Data)
				if !ok {
					setMetaReply(m, "set data reply failed. ", -1, getDuration(stime))
					fmt.Println(replymeta, replymeta.Info)
					stream.Send(replymeta)
					return errors.New("set data reply failed.")
				}
				setDataReply(d, string(line))
				stream.Send(datareply)
			}
		}
	})

	err = eg.Wait()
	log.Println("Stop operation handle. error:" + err.Error())
	return err
}
func (s *server) GetPullImageLog(req *pb.GetPullImageLogRequest, resp pb.DockerHandle_GetPullImageLogServer) error {
	return status.Errorf(codes.Unimplemented, "method GetPullImageLog not implemented")
}
func (s *server) GetContainerLog(req *pb.GetContainerRequest, resp pb.DockerHandle_GetContainerLogServer) error {
	return status.Errorf(codes.Unimplemented, "method GetContainerLog not implemented")
}

func Start(ctx context.Context, e *errgroup.Group, port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Printf("failed to listen: %v\n", err)
		return err
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterDockerHandleServer(grpcServer, &server{ctx: ctx, cm: container.GetCM()})

	e.Go(func() error {
		log.Println("grpc server is running... port:" + strconv.Itoa(port))
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("grpc server is stopped Ungracefully. %v", err)
			return err
		} else {
			log.Printf("grpc server is stopped. %v", err)
			return nil
		}
	})

	e.Go(func() error {
		<-ctx.Done()
		log.Println("grpc_server is shutting down now")
		grpcServer.Stop()
		return nil
	})

	return nil
}
