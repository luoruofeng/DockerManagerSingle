package container

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/crypto/ssh/terminal"
)

func TestConnContainer(t *testing.T) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal("Docker client init failed. " + err.Error())
	}
	InitContainerManager(context.Background(), cli)

	GetCM().DeleteContainerById("testconn")

	envs := make([]string, 0)
	envs = append(envs, "POSTGRES_PASSWORD=abc")
	createCreatedBody, err := GetCM().CreateContainer("e270a11b9c8a", envs, nil, nil, "testconn")
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println(createCreatedBody)
	}

	time.Sleep(time.Second * 4)

	waiter, err := GetCM().ConnContainer(createCreatedBody.ID)
	if err != nil {
		log.Fatal(err)
	} else {
		defer waiter.Close()

		go io.Copy(os.Stdout, waiter.Reader)
		go io.Copy(os.Stderr, waiter.Reader)
		go io.Copy(waiter.Conn, os.Stdin)
		// go io.Copy(os.Stdout, r)

		// 	n, err = c.Write([]byte("touch abc.txt \n"))
		// 	if err != nil {
		// 		fmt.Println(err)
		// 	}
		// 	fmt.Println(n)

		// 	c.Write([]byte("pwd \n"))

		// 	c.Write([]byte("cd / \n"))

		// 	c.Write([]byte("pwd \n"))

		// 	c.Write([]byte("echo 123"))
		// }()

		fd := int(os.Stdin.Fd())
		var oldState *terminal.State
		if terminal.IsTerminal(fd) {
			oldState, err = terminal.MakeRaw(fd)
			if err != nil { // TODO handle error?}

				go func() {
					for {
						consoleReader := bufio.NewReaderSize(os.Stdin, 1)
						input, _ := consoleReader.ReadByte() // Ctrl-C = 3
						if input == 3 {
							GetCM().DeleteContainerById(createCreatedBody.ID)
						}
						waiter.Conn.Write([]byte{input})
					}
				}()
			}

			statusCh, errCh := cli.ContainerWait(context.Background(), createCreatedBody.ID, dockercontainer.WaitConditionNotRunning)
			select {
			case err := <-errCh:
				if err != nil {
					panic(err)
				}
			case <-statusCh:
				if terminal.IsTerminal(fd) {
					terminal.Restore(fd, oldState)
				}
			}
			fmt.Println("EXIT")
		}
	}
}
