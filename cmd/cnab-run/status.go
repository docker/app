package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/opts"
	swarmtypes "github.com/docker/docker/api/types/swarm"
	"github.com/pkg/errors"
)

func statusAction(instanceName string) error {
	cli, err := getCli()
	if err != nil {
		return err
	}
	services, _ := runningServices(cli, instanceName)
	fmt.Fprintln(cli.Out(), services)
	return nil
}

func statusJSONAction(instanceName string) error {
	cli, err := getCli()
	if err != nil {
		return err
	}
	services, _ := runningServices(cli, instanceName)
	js, err := json.MarshalIndent(services, "", "    ")
	if err != nil {
		return err
	}
	fmt.Fprintln(cli.Out(), string(js))
	return nil
}

func getCli() (command.Cli, error) {
	cli, err := setupDockerContext()
	if err != nil {
		return nil, errors.Wrap(err, "unable to restore docker context")
	}
	return cli, nil
}

func runningServices(cli command.Cli, instanceName string) ([]swarmtypes.Service, error) {
	orchestratorRaw := os.Getenv(internal.DockerStackOrchestratorEnvVar)
	orchestrator, err := cli.StackOrchestrator(orchestratorRaw)
	if err != nil {
		return nil, err
	}
	return stack.GetServices(cli, getFlagset(orchestrator), orchestrator, options.Services{
		Filter:    opts.NewFilterOpt(),
		Namespace: instanceName,
	})
}
