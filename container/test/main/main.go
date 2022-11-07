package main

import (
	"context"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/docker/client"
	c "github.com/luoruofeng/dockermanagersingle/container"
)

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal("Docker client init failed. " + err.Error())
	}
	c.InitContainerManager(context.Background(), cli)

	respId, _ := c.GetCM().BashContainer("2c9b79533585e5b124ba10c49b76adee7df0ac525965bad50b8ad6c9859871b6")
	defer respId.Close()

	go io.Copy(os.Stdout, respId.Reader)
	go io.Copy(os.Stderr, respId.Reader)
	go io.Copy(respId.Conn, os.Stdin)

	time.Sleep(time.Second * 100)

}
