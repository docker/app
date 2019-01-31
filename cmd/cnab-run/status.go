package main

import (
	"os"

	"github.com/docker/cli/cli/command/stack"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/opts"
	"github.com/pkg/errors"
)

func status(instanceName string) error {
	cli, err := setupDockerContext()
	if err != nil {
		return errors.Wrap(err, "unable to restore docker context")
	}
	orchestratorRaw := os.Getenv(envVarOchestrator)
	orchestrator, err := cli.StackOrchestrator(orchestratorRaw)
	if err != nil {
		return err
	}
	return stack.RunServices(cli, getFlagset(orchestrator), orchestrator, options.Services{
		Namespace: instanceName,
		Filter:    opts.NewFilterOpt(),
	})
}
