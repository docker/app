package main

import (
	"context"
	"os"

	"github.com/docker/app/internal/com"

	"github.com/docker/cli/cli/command"
	"github.com/sirupsen/logrus"
)

func main() {
	fs, streams, session, err := com.ConnectToFront(os.Stdin, os.Stdout)
	if err != nil {
		panic(err)
	}
	// Set terminal emulation based on platform as required.

	logrus.SetOutput(streams.Err)

	dockerCli := command.NewDockerCli(streams.In, streams.Out, streams.Err, false)
	cmd := newRootCmd(dockerCli, fs)
	cmd.SetOutput(streams.Err)
	err = cmd.Execute()
	if vm, ok := err.(*com.VersionMismatch); ok {
		session.BackendVersionMismatch(context.Background(), vm)
	}
	com.Shutdown(context.Background(), session, streams)
	if err != nil {
		os.Exit(1)
	}
}
