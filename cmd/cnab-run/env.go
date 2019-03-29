package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/cli/cli/command"
	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/context/docker"
	kubcontext "github.com/docker/cli/cli/context/kubernetes"
	contextstore "github.com/docker/cli/cli/context/store"
	cliflags "github.com/docker/cli/cli/flags"
)

var storeConfig = contextstore.NewConfig(
	func() interface{} { return &command.DockerContext{} },
	contextstore.EndpointTypeGetter(docker.DockerEndpoint, func() interface{} { return &docker.EndpointMeta{} }),
	contextstore.EndpointTypeGetter(kubcontext.KubernetesEndpoint, func() interface{} { return &kubcontext.EndpointMeta{} }),
)

func setupDockerContext() (command.Cli, error) {
	s := contextstore.New(cliconfig.ContextStoreDir(), storeConfig)
	f, err := os.Open(internal.CredentialDockerContextPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := contextstore.Import("cnab", s, f); err != nil {
		return nil, err
	}
	cli, err := command.NewDockerCli()
	if err != nil {
		return nil, err
	}
	if err := cli.Initialize(&cliflags.ClientOptions{
		Common: &cliflags.CommonOptions{
			Context: "cnab",
		},
	}); err != nil {
		return nil, err
	}
	authConfigsJSON, err := ioutil.ReadFile(internal.CredentialRegistryPath)
	if err != nil {
		return nil, err
	}

	configFile := cli.ConfigFile()

	if err := json.Unmarshal(authConfigsJSON, &configFile.AuthConfigs); err != nil {
		return nil, err
	}

	return cli, nil
}
