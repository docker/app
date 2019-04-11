package packager

import (
	"fmt"
	"path"
	"strings"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/compose"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli/compose/loader"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
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
			DefaultValue: "",
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
			DataType:     "string",
			DefaultValue: "",
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
			DefaultValue: "yaml",
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
			DefaultValue: false,
		},
	}
	for name, envVar := range mapping.ParameterToCNABEnv {
		parameters[name] = bundle.ParameterDefinition{
			DataType: "string",
			Destination: &bundle.Location{
				EnvironmentVariable: envVar,
			},
			DefaultValue: flatParameters[name],
		}
	}
	autoParams, err := generateOverrideParameters(app.Composes())
	if err != nil {
		return nil, fmt.Errorf("unable to generate automatic parameters: %s", err)
	}
	for k, v := range autoParams {
		if _, exist := parameters[k]; !exist {
			parameters[k] = v
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

func generateOverrideParameters(composeFiles [][]byte) (map[string]bundle.ParameterDefinition, error) {

	merged := make(map[string]interface{})
	for _, composeFile := range composeFiles {
		parsed, err := loader.ParseYAML(composeFile)
		if err != nil {
			return nil, err
		}
		if err := mergo.Merge(&merged, parsed, mergo.WithAppendSlice, mergo.WithOverride); err != nil {
			return nil, err
		}
	}
	servicesRaw, ok := merged["services"]
	if !ok {
		return nil, nil
	}
	services, ok := servicesRaw.(map[string]interface{})
	if !ok {
		return nil, errors.New("unrecognized services type")
	}
	defs := make(map[string]bundle.ParameterDefinition)
	for serviceName, serviceValue := range services {
		serviceDef, ok := serviceValue.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("unerecognized type for service %q", serviceName)
		}
		addServiceOverrideParameters(serviceName, serviceDef, defs)
	}
	return defs, nil
}

func addServiceOverrideParameters(serviceName string, serviceDef map[string]interface{}, into map[string]bundle.ParameterDefinition) {
	for _, p := range serviceParametersToGenerate {
		pathParts := strings.Split(p, ".")
		if !hasKey(serviceDef, pathParts...) {
			dest := path.Join(internal.ComposeOverridesDir, "services", serviceName, strings.Join(pathParts, "/"))
			name := "services." + serviceName + "." + p
			into[name] = bundle.ParameterDefinition{
				DataType: "string",
				Destination: &bundle.Location{
					Path: dest,
				},
			}
		}
	}
}

var serviceParametersToGenerate = []string{
	"deploy.replicas",
	"deploy.resources.limits.cpus",
	"deploy.resources.limits.memory",
	"deploy.resources.reservations.cpus",
	"deploy.resources.reservations.memory",
}

func hasKey(source map[string]interface{}, path ...string) bool {
	if len(path) == 0 {
		return true
	}
	key, remaining := path[0], path[1:]
	subRaw, ok := source[key]
	if !ok {
		return false
	}
	if len(remaining) == 0 {
		return true
	}
	sub, ok := subRaw.(map[string]interface{})
	if !ok {
		return false
	}
	return hasKey(sub, remaining...)
}
