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
	log.Println("Start: Operation handle.")

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
	log.Printf("Get container connection.\n")
	if err != nil {
		log.Println(err.Error())
		setMetaReply(m, "bash container failed. "+err.Error(), -1, getDuration(stime))
		stream.Send(replymeta)
		return err
	}
	defer func() {
		log.Println("Close container connection.")
		hijack.Close()
	}()

	//get cmd line from grpc caller and send to hijack
	eg, ctx := errgroup.WithContext(context.Background())

	eg.Go(func() error {
		defer func() {
			log.Println("Exit: CMD reader")
		}()
		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				recvdata, err = stream.Recv()
				if err != nil {
					if err == io.EOF {
						setMetaReply(m, "", 1, getDuration(stime))
						stream.Send(replymeta)
						log.Println("Quit reason: Got EOF")
						return err
					} else {
						setMetaReply(m, err.Error(), -1, 0)
						stream.Send(replymeta)
						return err
					}
				}
				rbs := recvdata.GetData()
				rs := string(rbs)
				if strings.Trim(strings.ToLower(rs), "\n") == "exit" {
					log.Println("Quit reason: Read EXIT from client")
					setMetaReply(m, "", 1, getDuration(stime))
					stream.Send(replymeta)
					return errors.New("Quit reason: User send exit")
				}
				_, err = hijack.Conn.Write(rbs)
				if err != nil {
					m, _ := replymeta.Info.(*pb.DialogueReply_Meta)
					dt := int32(time.Since(stime)/time.Millisecond) / 1000
					setMetaReply(m, "Quit reason: hijack write content failed. content:"+rs+" error:"+err.Error(), -1, dt)
					stream.Send(replymeta)
					return err
				}
			}
		}
	})

	containerMesChan := make(chan []byte, 10000)
	containerErrChan := make(chan error, 10000)
	isContainerChanClosed := false

	defer func() {
		isContainerChanClosed = true
		close(containerMesChan)
		close(containerErrChan)
	}()

	//read container data from hijact to containerMesChan.ensure this func quit successfully when parent func quit.
	go func() {
		defer func() {
			log.Println("Exit: Container reader.")
		}()
		bufReader := bufio.NewReader(hijack.Reader)
		for {
			line, _, err := bufReader.ReadLine()
			if err != nil {
				containerErrChan <- err
			} else {
				if isContainerChanClosed {
					return
				}
				containerMesChan <- line
			}
		}
	}()

	//Write containerMesChan's content to grpc caller
	eg.Go(func() error {
		defer func() {
			log.Println("Exit: Writer")
		}()
		for {
			select {
			case <-ctx.Done():
				return nil
			case line := <-containerMesChan:

				datareply := createDataDialogueReply()
				d, ok := datareply.Info.(*pb.DialogueReply_Data)
				if !ok {
					setMetaReply(m, "set data reply failed. ", -1, getDuration(stime))
					log.Println(replymeta, replymeta.Info)
					stream.Send(replymeta)
					return errors.New("set data reply failed.")
				}
				setDataReply(d, string(line))
				stream.Send(datareply)
			case err := <-containerErrChan:
				if err != nil {
					if err == io.EOF {
						setMetaReply(m, "", 1, getDuration(stime))
						stream.Send(replymeta)
						log.Println("Got container EOF.")
						return err
					} else {
						setMetaReply(m, "container read content failed.  error:"+err.Error(), -1, getDuration(stime))
						stream.Send(replymeta)
						return err
					}
				}
			}
		}
	})

	err = eg.Wait()
	log.Println("Exit: operation handle. error:" + err.Error())
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