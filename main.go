package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	cli, err := client.NewClientWithOpts(client.WithVersion("1.37"))
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	for {
		if err := start(cli); err != nil {
			log.Println(err)
			start(cli)
		}
	}

}

func start(cli *client.Client) (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgChan, errChan := cli.Events(ctx, types.EventsOptions{})

L:
	for {
		select {
		case msg := <-msgChan:
			switch msg.Status {
			case "start":
				log.Println(msg)
			case "die":
				log.Println(msg)
				reader, err := cli.ContainerLogs(ctx, msg.ID, types.ContainerLogsOptions{
					Since:      "30s",
					ShowStdout: true,
					ShowStderr: true,
				})
				if err != nil {
					log.Println(err)
				}
				_, err = io.Copy(os.Stdout, reader)
				if err != nil && err != io.EOF {
					log.Fatal(err)
				}
			}
		case err = <-errChan:
			break L
		}
	}
	return
}
