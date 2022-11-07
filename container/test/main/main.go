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

	c.GetCM().DeleteContainerById("testconn")

	envs := make([]string, 0)
	envs = append(envs, "POSTGRES_PASSWORD=abc")
	createCreatedBody, err := c.GetCM().CreateContainer("e270a11b9c8a", envs, nil, nil, "testconn")
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println(createCreatedBody)
	}

	time.Sleep(time.Second * 4)

	c.GetCM().StartContainer(createCreatedBody.ID)
	time.Sleep(time.Second * 4)

	respId, _ := c.GetCM().BashContainer(createCreatedBody.ID)
	defer respId.Close()

	go io.Copy(os.Stdout, respId.Reader)
	go io.Copy(os.Stderr, respId.Reader)
	go io.Copy(respId.Conn, os.Stdin)

	time.Sleep(time.Second * 100)

}
