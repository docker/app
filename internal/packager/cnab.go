package packager

import (
	"encoding/json"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/bundle/definition"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/compose"
	"github.com/docker/app/types"
	"github.com/sirupsen/logrus"
)

const (
	// CNABVersion1_0_0 is the CNAB Schema version 1.0.0
	CNABVersion1_0_0 = "v1.0.0"
)

// DockerAppArgs represent the object passed to the invocation image
// by Docker App.
type DockerAppArgs struct {
	// Labels are the labels to add to containers on run
	Labels map[string]string `json:"labels,omitempty"`
}

// ToCNAB creates a CNAB bundle from an app package
func ToCNAB(app *types.App, invocationImageName string) (*bundle.Bundle, error) {
	mapping := ExtractCNABParameterMapping(app.Parameters())
	flatParameters := app.Parameters().Flatten()
	definitions := definition.Definitions{
		internal.ParameterArgs: {
			Type:        "string",
			Default:     "",
			Title:       "Arguments",
			Description: "Arguments that are passed by file to the invocation image",
		},
		internal.ParameterOrchestratorName: {
			Type: "string",
			Enum: []interface{}{
				"",
				"swarm",
				"kubernetes",
			},
			Default:     "",
			Title:       "Orchestrator",
			Description: "Orchestrator on which to deploy",
		},
		internal.ParameterKubernetesNamespaceName: {
			Type:        "string",
			Default:     "",
			Title:       "Namespace",
			Description: "Namespace in which to deploy",
		},
		internal.ParameterRenderFormatName: {
			Type: "string",
			Enum: []interface{}{
				"yaml",
				"json",
			},
			Default:     "yaml",
			Title:       "Render format",
			Description: "Output format for the render command",
		},
		internal.ParameterInspectFormatName: {
			Type: "string",
			Enum: []interface{}{
				"json",
				"pretty",
			},
			Default:     "json",
			Title:       "Inspect format",
			Description: "Output format for the inspect command",
		},
		internal.ParameterShareRegistryCredsName: {
			Type:        "boolean",
			Default:     false,
			Title:       "Share registry credentials",
			Description: "Share registry credentials with the invocation image",
		},
	}
	parameters := map[string]bundle.Parameter{
		internal.ParameterArgs: {
			Destination: &bundle.Location{
				Path: internal.DockerArgsPath,
			},
			ApplyTo: []string{
				"install",
				"upgrade",
			},
			Definition: internal.ParameterArgs,
		},
		internal.ParameterOrchestratorName: {
			Destination: &bundle.Location{
				EnvironmentVariable: internal.DockerStackOrchestratorEnvVar,
			},
			ApplyTo: []string{
				"install",
				"upgrade",
				"uninstall",
				internal.ActionStatusName,
			},
			Definition: internal.ParameterOrchestratorName,
		},
		internal.ParameterKubernetesNamespaceName: {
			Destination: &bundle.Location{
				EnvironmentVariable: internal.DockerKubernetesNamespaceEnvVar,
			},
			ApplyTo: []string{
				"install",
				"upgrade",
				"uninstall",
				internal.ActionStatusName,
			},
			Definition: internal.ParameterKubernetesNamespaceName,
		},
		internal.ParameterRenderFormatName: {
			Destination: &bundle.Location{
				EnvironmentVariable: internal.DockerRenderFormatEnvVar,
			},
			ApplyTo: []string{
				internal.ActionRenderName,
			},
			Definition: internal.ParameterRenderFormatName,
		},
		internal.ParameterInspectFormatName: {
			Destination: &bundle.Location{
				EnvironmentVariable: internal.DockerInspectFormatEnvVar,
			},
			ApplyTo: []string{
				internal.ActionInspectName,
			},
			Definition: internal.ParameterInspectFormatName,
		},
		internal.ParameterShareRegistryCredsName: {
			Destination: &bundle.Location{
				EnvironmentVariable: "DOCKER_SHARE_REGISTRY_CREDS",
			},
			Definition: internal.ParameterShareRegistryCredsName,
		},
	}
	for name, envVar := range mapping.ParameterToCNABEnv {
		definitions[name] = &definition.Schema{
			Type:    "string",
			Default: flatParameters[name],
		}
		parameters[name] = bundle.Parameter{
			Destination: &bundle.Location{
				EnvironmentVariable: envVar,
			},
			Definition: name,
		}
	}
	var maintainers []bundle.Maintainer
	for _, m := range app.Metadata().Maintainers {
		maintainers = append(maintainers, bundle.Maintainer{
			Email: m.Email,
			Name:  m.Name,
		})
	}

	bundleImages, err := extractBundleImages(app.Composes())
	if err != nil {
		return nil, err
	}

	payload, err := newCustomPayload()
	if err != nil {
		return nil, err
	}

	bndl := &bundle.Bundle{
		SchemaVersion: CNABVersion1_0_0,
		Custom: map[string]interface{}{
			internal.CustomDockerAppName: DockerAppCustom{
				Version: DockerAppPayloadVersionCurrent,
				Payload: payload,
			},
		},
		Credentials: map[string]bundle.Credential{
			internal.CredentialDockerContextName: {
				Location: bundle.Location{
					Path: internal.CredentialDockerContextPath,
				},
			},
			internal.CredentialRegistryName: {
				Location: bundle.Location{
					Path: internal.CredentialRegistryPath,
				},
			},
		},
		Description: app.Metadata().Description,
		InvocationImages: []bundle.InvocationImage{
			{
				BaseImage: bundle.BaseImage{
					Image:     invocationImageName,
					ImageType: "docker",
				},
			},
		},
		Maintainers: maintainers,
		Name:        app.Metadata().Name,
		Version:     app.Metadata().Version,
		Parameters:  parameters,
		Definitions: definitions,
		Actions: map[string]bundle.Action{
			internal.ActionInspectName: {
				Modifies:  false,
				Stateless: true,
			},
			internal.ActionRenderName: {
				Modifies:  false,
				Stateless: true,
			},
			internal.ActionStatusName: {
				Modifies: false,
			},
			internal.ActionStatusJSONName: {
				Modifies: false,
			},
		},
		Images: bundleImages,
	}

	if js, err := json.Marshal(bndl); err == nil {
		logrus.Debugf("App converted to CNAB %q", string(js))
	}

	return bndl, nil
}

func extractBundleImages(composeFiles [][]byte) (map[string]bundle.Image, error) {
	_, images, err := compose.Load(composeFiles)
	if err != nil {
		return nil, err
	}

	bundleImages := map[string]bundle.Image{}
	for serviceName, imageName := range images {
		bundleImages[serviceName] = bundle.Image{
			Description: imageName,
			BaseImage: bundle.BaseImage{
				Image:     imageName,
				ImageType: "docker",
			},
		}
	}
	return bundleImages, nil
}
