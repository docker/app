package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

func installAction(instanceName string) error {
	cli, err := setupDockerContext()
	if err != nil {
		return errors.Wrap(err, "unable to restore docker context")
	}
	app, err := packager.Extract("")
	// todo: merge additional compose file
	if err != nil {
		return err
	}
	defer app.Cleanup()

	orchestratorRaw := os.Getenv(internal.DockerStackOrchestratorEnvVar)
	orchestrator, err := cli.StackOrchestrator(orchestratorRaw)
	if err != nil {
		return err
	}
	bndl, err := getBundle()
	if err != nil {
		return err
	}
	parameters := packager.ExtractCNABParametersValues(packager.ExtractCNABParameterMapping(app.Parameters()), os.Environ())
	rendered, err := render.Render(app, parameters, bndl.Images)
	if err != nil {
		return err
	}
	if err = addLabels(rendered); err != nil {
		return err
	}
	addAppLabels(rendered, instanceName)

	if err := os.Chdir(app.Path); err != nil {
		return err
	}
	sendRegistryAuth, err := strconv.ParseBool(os.Getenv("DOCKER_SHARE_REGISTRY_CREDS"))
	if err != nil {
		return err
	}
	// todo: pass registry auth to invocation image
	return stack.RunDeploy(cli, getFlagset(orchestrator), rendered, orchestrator, options.Deploy{
		Namespace:        instanceName,
		ResolveImage:     swarm.ResolveImageAlways,
		SendRegistryAuth: sendRegistryAuth,
	})
}

func getFlagset(orchestrator command.Orchestrator) *pflag.FlagSet {
	result := pflag.NewFlagSet("", pflag.ContinueOnError)
	if orchestrator == command.OrchestratorKubernetes {
		result.String("namespace", os.Getenv(internal.DockerKubernetesNamespaceEnvVar), "")
	}
	return result
}

func addLabels(rendered *composetypes.Config) error {
	args, err := ioutil.ReadFile(internal.DockerArgsPath)
	if err != nil {
		return err
	}
	var a packager.DockerAppArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return err
	}
	for k, v := range a.Labels {
		for i, service := range rendered.Services {
			if service.Labels == nil {
				service.Labels = map[string]string{}
			}
			service.Labels[k] = v
			rendered.Services[i] = service
		}
	}
	return nil
}

func addAppLabels(rendered *composetypes.Config, instanceName string) {
	for i, service := range rendered.Services {
		if service.Labels == nil {
			service.Labels = map[string]string{}
		}
		service.Labels[internal.LabelAppNamespace] = instanceName
		service.Labels[internal.LabelAppVersion] = internal.Version
		rendered.Services[i] = service
	}
}
