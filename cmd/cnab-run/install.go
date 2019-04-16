package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	yaml "gopkg.in/yaml.v2"
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
	overrides, err := parseOverrides()
	if err != nil {
		return errors.Wrap(err, "unable to parse auto-parameter values")
	}
	app, err := packager.Extract("", types.WithComposes(bytes.NewReader(overrides)))
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

func parseOverrides() ([]byte, error) {
	root := make(map[string]interface{})
	if err := filepath.Walk(internal.ComposeOverridesDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() && fi.Size() > 0 {
			bytes, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(internal.ComposeOverridesDir, path)
			if err != nil {
				return err
			}
			splitPath := strings.Split(rel, "/")
			if err := setValue(root, splitPath, string(bytes)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return yaml.Marshal(root)
}

func setValue(root map[string]interface{}, path []string, value string) error {
	key, sub := path[0], path[1:]
	if len(sub) == 0 {
		converted, err := converterFor(key)(value)
		if err != nil {
			return err
		}
		root[key] = converted
		return nil
	}
	subMap := make(map[string]interface{})
	root[key] = subMap
	return setValue(subMap, sub, value)
}

type valueConverter func(string) (interface{}, error)

func stringConverter(v string) (interface{}, error) {
	return v, nil
}

func intConverter(v string) (interface{}, error) {
	return strconv.ParseInt(v, 10, 32)
}

func converterFor(key string) valueConverter {
	switch key {
	case "replicas":
		return intConverter
	default:
		return stringConverter
	}
}
