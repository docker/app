package build

import (
	"fmt"
	"path"

	"github.com/docker/app/types"
	"github.com/docker/buildx/build"
	"github.com/docker/cli/cli/compose/loader"
	compose "github.com/docker/cli/cli/compose/types"
)

// parseCompose do parse app compose file and extract buildx Options
// We don't rely on bake's ReadTargets + TargetsToBuildOpt here as we have to skip environment variable interpolation
func parseCompose(app *types.App, contextPath string, options buildOptions) (map[string]build.Options, []ServiceConfig, error) {
	parsed, err := loader.ParseYAML(app.Composes()[0])
	if err != nil {
		return nil, nil, err
	}

	services, err := load(parsed, options.args)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to parse compose file: %s", err)
	}

	pulledServices := []ServiceConfig{}
	opts := map[string]build.Options{}
	for _, service := range services {
		if service.Build == nil {
			pulledServices = append(pulledServices, service)
			continue
		}
		var tags []string
		if service.Image != nil {
			tags = append(tags, *service.Image)
		}

		if service.Build.Dockerfile == "" {
			service.Build.Dockerfile = "Dockerfile"
		}
		opts[service.Name] = build.Options{
			Inputs: build.Inputs{
				ContextPath:    path.Join(contextPath, service.Build.Context),
				DockerfilePath: path.Join(contextPath, service.Build.Context, service.Build.Dockerfile),
			},
			BuildArgs: flatten(service.Build.Args),
			NoCache:   options.noCache,
			Pull:      options.pull,
			Tags:      tags,
		}
	}
	return opts, pulledServices, nil
}

func flatten(in compose.MappingWithEquals) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string)
	for k, v := range in {
		if v == nil {
			continue
		}
		out[k] = *v
	}
	return out
}
