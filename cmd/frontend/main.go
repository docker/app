package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/com"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	protobuf "github.com/gogo/protobuf/types"
)

type frontendServerImpl struct {
}

func (frontendServerImpl) FileContent(path *protobuf.StringValue, chunkSink com.FrontService_FileContentServer) error {
	f, err := os.Open(path.Value)
	if err != nil {
		return err
	}
	defer f.Close()
	buffer := make([]byte, 4096)
	for {
		read, err := f.Read(buffer)
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		}
		fmt.Printf("received %d bytes\n", read)
		if err = chunkSink.Send(&protobuf.BytesValue{Value: buffer[:read]}); err != nil {
			return err
		}
	}
}
func (frontendServerImpl) FileList(path *protobuf.StringValue, statSink com.FrontService_FileListServer) error {
	fmt.Println("Received file list")
	stats, err := ioutil.ReadDir(path.Value)
	if err != nil {
		return err
	}
	for _, stat := range stats {
		if err = statSink.Send(&com.FileStat{
			IsDir: stat.IsDir(),
			Mode:  int32(stat.Mode()),
			Name:  stat.Name(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	args := os.Args[1:]
	dockerCli, err := dclient.NewClientWithOpts(dclient.FromEnv)
	if err != nil {
		panic(err)
	}
	dockerCli.NegotiateAPIVersion(context.Background())
	cont, err := dockerCli.ContainerCreate(context.Background(), &container.Config{
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		Tty:          false,
		StdinOnce:    true,
		Cmd:          args,
		Image:        "docker/app-backend:" + internal.Version,
	}, &container.HostConfig{
		AutoRemove: false,
	}, nil, "docker-app")
	if err != nil {
		panic(err)
	}

	attach, err := dockerCli.ContainerAttach(context.Background(), cont.ID, types.ContainerAttachOptions{
		Logs:   false,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Stream: true,
	})
	if err != nil {
		panic(err)
	}
	defer attach.Close()
	outReader, outWriter := io.Pipe()
	go func() {

		written, err := stdcopy.StdCopy(outWriter, os.Stderr, attach.Reader)
		fmt.Printf("written: %d, err: %s\n", written, err)
	}()

	go func() {
		com.RunFrontService(frontendServerImpl{}, outReader, attach.Conn, os.Stdin, os.Stdout, os.Stderr)
	}()

	err = dockerCli.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		panic(err)
	}
	waitOk, waitErr := dockerCli.ContainerWait(context.Background(), cont.ID, container.WaitConditionNextExit)
	select {
	case <-waitOk:
		return
	case err = <-waitErr:
		panic(err)
	}

}
