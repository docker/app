package main

import (
	"os"

	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/pkg/term"
	"github.com/sirupsen/logrus"
)

func main() {
	// Set terminal emulation based on platform as required.
	stdin, stdout, stderr := term.StdStreams()
	logrus.SetOutput(stderr)

	dockerCli := command.NewDockerCli(stdin, stdout, stderr, false, nil)
	cmd := newRootCmd(dockerCli)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
