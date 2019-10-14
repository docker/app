package build

import (
	"errors"
	"fmt"
	"path"

	"github.com/docker/distribution/reference"

	"github.com/docker/app/types"
	"github.com/docker/buildx/build"
	"github.com/docker/cli/cli/compose/loader"
	compose "github.com/docker/cli/cli/compose/types"
)

// parseCompose do parse app compose file and extract buildx Options
// We don't rely on bake's ReadTargets + TargetsToBuildOpt here as we have to skip environment variable interpolation
func parseCompose(app *types.App, contextPath string, options buildOptions, reference reference.Reference) (map[string]build.Options, error) {
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

		if service.Name == "installer" {
			return nil, errors.New("'installer' is a reserved service name, please fix your docker-compose.yml file")
		}

		tags := []string{fmt.Sprintf("%s-%s", reference.String(), service.Name)}

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
