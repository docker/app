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
func parseCompose(app *types.App, options buildOptions) (map[string]build.Options, error) {
	parsed, err := loader.ParseYAML(app.Composes()[0])
	if err != nil {
		return nil, err
	}

	services, err := load(parsed)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse compose file: %s", err)
	}

	opts := map[string]build.Options{}
	for _, service := range services {
		if service.Build == nil {
			continue
		}

		var tags []string
		if service.Image != nil && *service.Image != "" {
			tags = []string{*service.Image}
		}

		// FIXME docker app init should update relative paths
		// compose file has been copied to x.dockerapp, so the relative path to build context get broken
		contextPath := path.Join(app.Path, "..", service.Build.Context)
		if service.Build.Dockerfile == "" {
			service.Build.Dockerfile = "Dockerfile"
		}
		dockerfile := path.Join(contextPath, service.Build.Dockerfile)
		opts[service.Name] = build.Options{
			Inputs: build.Inputs{
				ContextPath:    contextPath,
				DockerfilePath: dockerfile,
			},
			BuildArgs: flatten(service.Build.Args),
			NoCache:   options.noCache,
			Pull:      options.pull,
			Tags:      tags,
		}
	}
	return opts, nil
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
