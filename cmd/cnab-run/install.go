package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

const (
	// imageMapFilePath is the path where the CNAB runtime will put the actual
	// service to image mapping to use
	imageMapFilePath = "/cnab/app/image-map.json"
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
	imageMap, err := getBundleImageMap()
	if err != nil {
		return err
	}
	parameters := packager.ExtractCNABParametersValues(packager.ExtractCNABParameterMapping(app.Parameters()), os.Environ())
	rendered, err := render.Render(app, parameters, imageMap)
	if err != nil {
		return err
	}
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

func getBundleImageMap() (map[string]bundle.Image, error) {
	mapJSON, err := ioutil.ReadFile(imageMapFilePath)
	if err != nil {
		return nil, err
	}
	var result map[string]bundle.Image
	if err := json.Unmarshal(mapJSON, &result); err != nil {
		return nil, err
	}
	return result, nil
}
