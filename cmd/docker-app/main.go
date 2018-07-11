package main

import (
	"os"

	"github.com/docker/app/internal/com"

	"github.com/docker/cli/cli/command"
	"github.com/sirupsen/logrus"
)

func main() {
	_, streams, err := com.ConnectToFront(os.Stdin, os.Stdout)
	if err != nil {
		panic(err)
	}
	// Set terminal emulation based on platform as required.

	logrus.SetOutput(streams.Err)

	dockerCli := command.NewDockerCli(streams.In, streams.Out, streams.Err, false)
	cmd := newRootCmd(dockerCli)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
