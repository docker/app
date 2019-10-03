package build

import (
	"fmt"
	"path"
	"reflect"

	"github.com/docker/app/types"
	"github.com/docker/buildx/build"
	"github.com/docker/cli/cli/compose/loader"
	compose "github.com/docker/cli/cli/compose/types"
)

// parseCompose do parse app compose file and extract buildx Options
// We don't rely on bake's ReadTargets + TargetsToBuildOpt here as we have to skip environment variable interpolation
func parseCompose(app *types.App, options buildOptions) (map[string]build.Options, error) {
	// Fixme can have > 1 composes ?
	parsed, err := loader.ParseYAML(app.Composes()[0])
	if err != nil {
		return nil, err
	}

	services, err := Load(parsed)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse compose file: %s", err)
	}

	var zeroBuildConfig ImageBuildConfig
	opts := map[string]build.Options{}
	for _, service := range services {
		if reflect.DeepEqual(service.Build, zeroBuildConfig) {
			continue
		}
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
