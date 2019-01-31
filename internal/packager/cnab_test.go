package packager

import (
	"testing"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/app/types"
	"gotest.tools/assert"
)

func TestToCNAB(t *testing.T) {
	app, err := types.NewAppFromDefaultFiles("testdata/packages/packing.dockerapp")
	assert.NilError(t, err)
	actual := ToCNAB(app, "test-image")
	expected := &bundle.Bundle{
		Description: "hello",
		Name:        "my-namespace/packing",
		Maintainers: []bundle.Maintainer{
			{Name: "bearclaw", Email: "bearclaw@bearclaw.com"},
			{Name: "bob", Email: "bob@bob.com"},
		},
		Version: "0.1.0",
		InvocationImages: []bundle.InvocationImage{
			{
				BaseImage: bundle.BaseImage{
					Image:     "test-image",
					ImageType: "docker",
				},
			},
		},
		Credentials: map[string]bundle.Location{
			"docker.context": {
				Path: "/cnab/app/context.dockercontext",
			},
		},
		Parameters: map[string]bundle.ParameterDefinition{
			"docker.orchestrator": {
				DataType: "string",
				AllowedValues: []interface{}{
					"",
					"swarm",
					"kubernetes",
				},
				Destination: &bundle.Location{
					EnvironmentVariable: "DOCKER_STACK_ORCHESTRATOR",
				},
				Metadata: bundle.ParameterMetadata{
					Description: "Orchestrator on which to deploy",
				},
				DefaultValue: "",
			},
			"docker.kubernetes-namespace": {
				DataType: "string",
				Destination: &bundle.Location{
					EnvironmentVariable: "DOCKER_KUBERNETES_NAMESPACE",
				},
				Metadata: bundle.ParameterMetadata{
					Description: "Namespace in which to deploy",
				},
				DefaultValue: "",
			},
			"watcher.image": {
				DataType: "string",
				Destination: &bundle.Location{
					EnvironmentVariable: "docker_param1",
				},
				DefaultValue: "watcher:latest",
			},
		},
		Actions: map[string]bundle.Action{
			"inspect": {
				Modifies: false,
			},
		},
	}
	assert.DeepEqual(t, actual, expected)
}
