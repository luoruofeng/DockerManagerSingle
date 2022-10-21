package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/luoruofeng/dockermanagersingle/api"
	"github.com/luoruofeng/dockermanagersingle/container"
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

	filePath, proxyPort, apiPort := ReadConfigFlag()

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

	log.Printf("args(proxyPort=%d apiPort=%d)", types.GConfig.ProxyPort, types.GConfig.ApiPort)

	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		log.Println("Docker client init failed. " + err.Error())
		cancel()
		goto ERR

	}
	container.InitContainerManager(ctx, cli)
	api.Start(ctx, e, types.GConfig.ApiPort, types.GConfig.ApiReadTimeout, types.GConfig.ApiWriteTimeout, types.GConfig.ApiIdleTimeout)
	// httpProxy.Start(ctx, cancel, types.GConfig.ProxyPort)

ERR:
	e.Wait()
	log.Println("DockerManagerSingle EXIT")
}

func ReadConfigFlag() (configFile string, proxyPort *int, apiPort *int) {
	configFile = *flag.String("config", "./config.yaml", "The configuration yaml file")
	proxyPort = flag.Int("proxy_port", 0, "The proxy url's port")
	apiPort = flag.Int("api_port", 0, "The api url's port")
	flag.Parse()
	return
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