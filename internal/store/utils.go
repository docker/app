package store

import (
	"os"

	"github.com/docker/cli/cli/command"

	"github.com/docker/app/internal"
)

func GetOrchestrator(cli command.Cli) (command.Orchestrator, error) {
	orchestratorRaw := os.Getenv(internal.DockerStackOrchestratorEnvVar)
	return cli.StackOrchestrator(orchestratorRaw)
}
