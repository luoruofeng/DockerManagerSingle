package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/luoruofeng/dockermanagersingle/api"
	"github.com/luoruofeng/dockermanagersingle/container"
	"github.com/luoruofeng/dockermanagersingle/grpc"
	"github.com/luoruofeng/dockermanagersingle/types"
	"golang.org/x/sync/errgroup"

	dockerclient "github.com/docker/docker/client"
	"gopkg.in/yaml.v3"
)

func main() {

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	e, ctx := errgroup.WithContext(ctx)

	isDev, filePath, proxyPort, apiPort, grpcPort, apiEnable, grpcEnable := ReadConfigFlag()
	logFile := setLogger(*isDev, "./log.txt")
	defer func() {
		err := logFile.Close()
		if err != nil {
			panic(err)
		}
	}()

	_, err := ReadConfig(filePath)
	if err != nil {
		cancel()
		fmt.Println(err)
		panic(err)
	}

	if *proxyPort != 0 {
		types.GConfig.ProxyPort = *proxyPort
	}

	if *apiPort != 0 {
		types.GConfig.ApiPort = *apiPort
	}

	if *grpcPort != 0 {
		types.GConfig.GrpcPort = *grpcPort
	}

	if !*grpcEnable {
		types.GConfig.GRPCEnable = *grpcEnable
	}

	if !*apiEnable {
		types.GConfig.APIEnable = *apiEnable
	}

	log.Printf("args(apiEnable=%v grpcEnable=%v proxyPort=%d apiPort=%d)\n", types.GConfig.APIEnable, types.GConfig.GRPCEnable, types.GConfig.ProxyPort, types.GConfig.ApiPort)

	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		log.Println("Docker client init failed. " + err.Error())
		cancel()
		goto ERR

	}
	container.InitContainerManager(ctx, cli)
	if types.GConfig.APIEnable {
		api.Start(ctx, e, types.GConfig.ApiPort, types.GConfig.ApiReadTimeout, types.GConfig.ApiWriteTimeout, types.GConfig.ApiIdleTimeout)
	}

	if types.GConfig.GRPCEnable {
		err = grpc.Start(ctx, e, types.GConfig.GrpcPort)
		if err != nil {
			cancel()
			goto ERR
		}
	}

	// httpProxy.Start(ctx, cancel, types.GConfig.ProxyPort)

ERR:
	e.Wait()
	log.Println("DockerManagerSingle EXIT")
}

func ReadConfigFlag() (idDev *bool, configFile string, proxyPort *int, apiPort *int, grpcPort *int, apiEnable *bool, grpcEnable *bool) {
	idDev = flag.Bool("dev", false, "log write to file and terminal")
	apiEnable = flag.Bool("api_enable", true, "API server is availability")
	grpcEnable = flag.Bool("grpc_enable", true, "GRPC server is availability")
	configFile = *flag.String("config", "./config.yaml", "The configuration yaml file")
	proxyPort = flag.Int("proxy_port", 0, "The proxy server url's port")
	grpcPort = flag.Int("grpc_port", 0, "The grpc server url's port")
	apiPort = flag.Int("api_port", 0, "The api server url's port")
	flag.Parse()
	return
}

func setLogger(isDev bool, path string) *os.File {
	var w io.Writer
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE, 0755)
	if isDev {
		w = io.MultiWriter(os.Stdout, f)
	}
	if err != nil {
		log.Fatal(err)
	}
	logger := log.Default()
	if w == nil {
		logger.SetOutput(f)
	} else {
		logger.SetOutput(w)
	}

	return f
}

func ReadConfig(filePath string) (*types.Config, error) {
	result := &types.Config{}

	b, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("config file is not exist")
		} else {
			return nil, err
		}
	}

	err = yaml.Unmarshal(b, result)
	if err != nil {
		return nil, err
	}
	types.GConfig = result
	return result, nil
}
