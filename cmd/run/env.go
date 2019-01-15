package main

import (
	"os"

	"github.com/docker/cli/cli/command"
	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/context/docker"
	kubcontext "github.com/docker/cli/cli/context/kubernetes"
	contextstore "github.com/docker/cli/cli/context/store"
	cliflags "github.com/docker/cli/cli/flags"
)

const (
	envVarOchestrator = "DOCKER_STACK_ORCHESTRATOR"
	fileDockerContext = "/cnab/app/context.dockercontext"
)

var storeConfig = contextstore.NewConfig(
	func() interface{} { return &command.DockerContext{} },
	contextstore.EndpointTypeGetter(docker.DockerEndpoint, func() interface{} { return &docker.EndpointMeta{} }),
	contextstore.EndpointTypeGetter(kubcontext.KubernetesEndpoint, func() interface{} { return &kubcontext.EndpointMeta{} }),
)

func setupDockerContext() (command.Cli, error) {
	s := contextstore.New(cliconfig.ContextStoreDir(), storeConfig)
	f, err := os.Open(fileDockerContext)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := contextstore.Import("cnab", s, f); err != nil {
		return nil, err
	}
	cli := command.NewDockerCli(os.Stdin, os.Stdout, os.Stderr, false, nil)
	return cli, cli.Initialize(&cliflags.ClientOptions{
		Common: &cliflags.CommonOptions{
			Context: "cnab",
		},
	})
}
