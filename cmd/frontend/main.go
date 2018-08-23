package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/app/internal"

	"github.com/docker/app/internal/com"
	"github.com/docker/app/internal/fs"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

func runBackend(version string) error {
	args := os.Args[1:]
	dockerCli, err := dclient.NewClientWithOpts(dclient.FromEnv)
	if err != nil {
		return err
	}
	dockerCli.NegotiateAPIVersion(context.Background())
	cont, err := dockerCli.ContainerCreate(context.Background(), &container.Config{
		AttachStderr: true,
		AttachStdin:  true,
		AttachStdout: true,
		Tty:          false,
		StdinOnce:    true,
		OpenStdin:    true,
		Cmd:          args,
		Image:        "docker/app-backend:" + version,
	}, &container.HostConfig{}, nil, "docker-app")
	if err != nil {
		return err
	}
	defer dockerCli.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{Force: true})

	attach, err := dockerCli.ContainerAttach(context.Background(), cont.ID, types.ContainerAttachOptions{
		Logs:   false,
		Stdin:  true,
		Stdout: true,
		Stderr: true,
		Stream: true,
	})
	if err != nil {
		return err
	}
	defer attach.Close()
	ended := make(chan error)
	outReader, outWriter := io.Pipe()
	go stdcopy.StdCopy(outWriter, os.Stderr, attach.Conn)
	go func() {
		ended <- com.RunFrontService(fs.FrontFileServer{}, outReader, attach.Conn, os.Stdin, os.Stdout, os.Stderr)
	}()
	err = dockerCli.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	err = <-ended
	if err != nil {
		return err
	}
	okChan, errChan := dockerCli.ContainerWait(context.Background(), cont.ID, container.WaitConditionNextExit)
	select {
	case <-okChan:
		return nil
	case err = <-errChan:
		return err
	}
}

func main() {
	version := os.Getenv("DOCKERAPP_VERSION")
	if version == "" {
		version = internal.Version
	}
	for {
		err := runBackend(version)
		if err == nil {
			break
		}
		if vm, ok := err.(*com.VersionMismatch); ok {
			fmt.Printf("Backend version mismatch. retrying with backend version %s\n", vm.PackageVersion)
			version = vm.PackageVersion
		} else {
			panic(err)
		}
	}
}
