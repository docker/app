package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/docker/cli/cli/command"
	cliconfig "github.com/docker/cli/cli/config"
	contextstore "github.com/docker/cli/cli/context/store"
	cliflags "github.com/docker/cli/cli/flags"
)

const (
	envVarOchestrator = "DOCKER_STACK_ORCHESTRATOR"
	fileDockerContext = "/cnab/app/context.dockercontext"
)

func setupDockerContext() (command.Cli, error) {
	s := contextstore.New(cliconfig.ContextStoreDir())
	f, err := os.Open(fileDockerContext)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if err := contextstore.Import("cnab", s, f); err != nil {
		return nil, err
	}
	cli := command.NewDockerCli(os.Stdin, os.Stdout, os.Stderr, false, nil)
	if err := cli.Initialize(&cliflags.ClientOptions{
		Common: &cliflags.CommonOptions{
			Context: "cnab",
		},
	}); err != nil {
		return nil, err
	}

	authConfigsJSON, err := ioutil.ReadFile("/cnab/app/registry-creds.json")
	if err != nil {
		return nil, err
	}

	configFile := cli.ConfigFile()

	if err := json.Unmarshal(authConfigsJSON, &configFile.AuthConfigs); err != nil {
		return nil, err
	}

	return cli, nil
}
