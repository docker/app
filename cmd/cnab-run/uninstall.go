package main

import (
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/cli/cli/command/stack"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/pkg/errors"
)

func uninstallAction(instanceName string) error {
	cli, err := setupDockerContext()
	if err != nil {
		return errors.Wrap(err, "unable to restore docker context")
	}
	orchestratorRaw := os.Getenv(internal.DockerStackOrchestratorEnvVar)
	orchestrator, err := cli.StackOrchestrator(orchestratorRaw)
	if err != nil {
		return err
	}
	return stack.RunRemove(cli, getFlagset(orchestrator), orchestrator, options.Remove{
		Namespaces: []string{instanceName},
	})
}
