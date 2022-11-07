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

	"github.com/docker/docker/api/types"
	"github.com/luoruofeng/dockermanagersingle/container"

	pb "github.com/luoruofeng/dockermanagersingle/pb"
	"golang.org/x/sync/errgroup"

	"google.golang.org/grpc"
)

const ReadByteLen int = 1 << 8

var sessionDuration time.Duration = 600

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
	bctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(sessionDuration))
	defer cancel()

	cmdMesChan := make(chan *pb.OperationRequest, 10000)
	cmdErrChan := make(chan error, 10000)
	isCmdChanClosed := false

	defer func() {
		isCmdChanClosed = true
		close(cmdErrChan)
		close(cmdMesChan)
	}()

	go func() {
		defer func() {
			log.Println("Exit: Cmd Chan Reader.")
		}()
		for {
			recvdata, err = stream.Recv()
			if err != nil {
				if isCmdChanClosed {
					return
				}
				cmdErrChan <- err
				return
			} else {
				if isCmdChanClosed {
					return
				}
				cmdMesChan <- recvdata
			}
		}
	}()

	eg, ctx := errgroup.WithContext(bctx)
	eg.Go(func() error {
		defer func() {
			log.Println("Exit: CMD Reader")
		}()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-cmdErrChan:
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
			case recvdata := <-cmdMesChan:
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

	containerMesChan := make(chan string, 10000)
	containerErrChan := make(chan error, 10000)
	isContainerChanClosed := false

	defer func() {
		isContainerChanClosed = true
		close(containerMesChan)
		close(containerErrChan)
	}()

	//read container data from hijact to containerMesChan.ensure this func quit successfully when parent func quit.
	go func(hijack *types.HijackedResponse) {
		defer func() {
			log.Println("Exit: Container Chan reader.")
		}()

		for {
			bs := make([]byte, ReadByteLen)
			n, err := hijack.Reader.Read(bs)
			if err != nil {
				if isContainerChanClosed {
					return
				}
				containerErrChan <- err
				return
			} else {
				if isContainerChanClosed {
					return
				}
				containerMesChan <- string(bs[:n])
			}
		}
	}(hijack)

	//Write containerMesChan's content to grpc caller
	eg.Go(func() error {
		defer func() {
			log.Println("Exit: CMD Writer")
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
	log.Println("Exit: operation handle.")
	return err
}
func (s *server) PullImageWithLog(req *pb.PullImageWithLogRequest, reply pb.DockerHandle_PullImageWithLogServer) error {
	log.Println("Start: Image pull handle.")
	stime := time.Now()
	mreply := createMetaDialogueReply()
	m, _ := mreply.Info.(*pb.DialogueReply_Meta)
	dreply := createDataDialogueReply()
	d, _ := dreply.Info.(*pb.DialogueReply_Data)

	rc, err := container.GetCM().PullImage(req.ImageName, req.ImageVersion)
	defer func() {
		err := rc.Close()
		if err != nil {
			log.Println(err)
		}
		log.Println("End: Image pull handle.")
	}()

	if err != nil {
		setMetaReply(m, err.Error(), -1, 0)
		reply.Send(mreply)
		return err
	}

	bufr := bufio.NewReader(rc)
	for {
		line, _, err := bufr.ReadLine()
		if err != nil {
			if err == io.EOF {
				setMetaReply(m, "EOF", 1, getDuration(stime))
				reply.Send(mreply)
				return err
			} else {
				log.Println(err)
				setMetaReply(m, err.Error(), -1, 0)
				reply.Send(mreply)
				return err
			}
		}
		setDataReply(d, string(line))
		err = reply.Send(dreply)
		if err != nil {
			log.Println(err)
			setMetaReply(m, err.Error(), -1, 0)
			reply.Send(mreply)
			return err
		}
	}
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
