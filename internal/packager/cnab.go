package packager

import (
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/compose"
	"github.com/docker/app/types"
)

// ToCNAB creates a CNAB bundle from an app package
func ToCNAB(app *types.App, invocationImageName string) (*bundle.Bundle, error) {
	mapping := ExtractCNABParameterMapping(app.Parameters())
	flatParameters := app.Parameters().Flatten()
	parameters := map[string]bundle.ParameterDefinition{
		internal.ParameterOrchestratorName: {
			DataType: "string",
			AllowedValues: []interface{}{
				"",
				"swarm",
				"kubernetes",
			},
			Default: "",
			Destination: &bundle.Location{
				EnvironmentVariable: internal.DockerStackOrchestratorEnvVar,
			},
			Metadata: &bundle.ParameterMetadata{
				Description: "Orchestrator on which to deploy",
			},
			ApplyTo: []string{
				"install",
				"upgrade",
				"uninstall",
				internal.ActionStatusName,
			},
		},
		internal.ParameterKubernetesNamespaceName: {
			DataType: "string",
			Default:  "",
			Destination: &bundle.Location{
				EnvironmentVariable: internal.DockerKubernetesNamespaceEnvVar,
			},
			Metadata: &bundle.ParameterMetadata{
				Description: "Namespace in which to deploy",
			},
			ApplyTo: []string{
				"install",
				"upgrade",
				"uninstall",
				internal.ActionStatusName,
			},
		},
		internal.ParameterRenderFormatName: {
			DataType: "string",
			AllowedValues: []interface{}{
				"yaml",
				"json",
			},
			Default: "yaml",
			Destination: &bundle.Location{
				EnvironmentVariable: internal.DockerRenderFormatEnvVar,
			},
			Metadata: &bundle.ParameterMetadata{
				Description: "Output format for the render command",
			},
			ApplyTo: []string{
				internal.ActionRenderName,
			},
		},
		internal.ParameterShareRegistryCredsName: {
			DataType: "bool",
			Destination: &bundle.Location{
				EnvironmentVariable: "DOCKER_SHARE_REGISTRY_CREDS",
			},
			Metadata: &bundle.ParameterMetadata{
				Description: "Share registry credentials with the invocation image",
			},
			Default: false,
		},
	}
	for name, envVar := range mapping.ParameterToCNABEnv {
		parameters[name] = bundle.ParameterDefinition{
			DataType: "string",
			Destination: &bundle.Location{
				EnvironmentVariable: envVar,
			},
			Default: flatParameters[name],
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

	return &bundle.Bundle{
		Credentials: map[string]bundle.Location{
			internal.CredentialDockerContextName: {
				Path: internal.CredentialDockerContextPath,
			},
			internal.CredentialRegistryName: {
				Path: internal.CredentialRegistryPath,
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
		},
		Images: bundleImages,
	}, nil
}

func extractBundleImages(composeFiles [][]byte) (map[string]bundle.Image, error) {
	_, images, err := compose.Load(composeFiles, func(v string) (string, error) { return v, nil })
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
