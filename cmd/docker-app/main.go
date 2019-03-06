package main

import (
	"fmt"
	"os"

	"github.com/docker/cli/cli/command"
	"github.com/sirupsen/logrus"
)

func main() {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	logrus.SetOutput(dockerCli.Err())
	cmd := newRootCmd(dockerCli)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
