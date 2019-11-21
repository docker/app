package build

import (
	"path"
	"strings"

	"github.com/docker/app/render"

	"github.com/docker/app/types"
	"github.com/docker/buildx/build"
	compose "github.com/docker/cli/cli/compose/types"
)

// parseCompose do parse app compose file and extract buildx Options
// We don't rely on bake's ReadTargets + TargetsToBuildOpt here as we have to skip environment variable interpolation
func parseCompose(app *types.App, contextPath string, options buildOptions) (map[string]build.Options, []compose.ServiceConfig, error) {
	comp, err := render.Render(app, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	buildArgs := buildArgsToMap(options.args)

	pulledServices := []compose.ServiceConfig{}
	opts := map[string]build.Options{}
	for _, service := range comp.Services {
		if service.Build.Context == "" {
			pulledServices = append(pulledServices, service)
			continue
		}
		var tags []string
		if service.Image != "" {
			tags = append(tags, service.Image)
		}

		if service.Build.Dockerfile == "" {
			service.Build.Dockerfile = "Dockerfile"
		}
		opts[service.Name] = build.Options{
			Inputs: build.Inputs{
				ContextPath:    path.Join(contextPath, service.Build.Context),
				DockerfilePath: path.Join(contextPath, service.Build.Context, service.Build.Dockerfile),
			},
			BuildArgs: flatten(mergeArgs(service.Build.Args, buildArgs)),
			NoCache:   options.noCache,
			Pull:      options.pull,
			Tags:      tags,
		}
	}
	return opts, pulledServices, nil
}

func buildArgsToMap(array []string) map[string]string {
	result := make(map[string]string)
	for _, value := range array {
		parts := strings.SplitN(value, "=", 2)
		key := parts[0]
		if len(parts) == 1 {
			result[key] = ""
		} else {
			result[key] = parts[1]
		}
	}
	return result
}

func mergeArgs(src compose.MappingWithEquals, values map[string]string) compose.MappingWithEquals {
	for key := range src {
		if val, ok := values[key]; ok {
			if val == "" {
				src[key] = nil
			} else {
				src[key] = &val
			}
		}
	}
	return src
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
