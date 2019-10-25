package build

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/docker/cli/cli/compose/loader"
	compose "github.com/docker/cli/cli/compose/types"
	"github.com/pkg/errors"
)

// A minimal subset of github.com/docker/cli/cli/compose/types/types.go for the purpose of loading the build configuration

type ServiceConfig struct {
	Name string `yaml:"-" json:"-"`

	Build *ImageBuildConfig
	Image *string
}

type ImageBuildConfig struct {
	Context    string                    `yaml:",omitempty" json:"context,omitempty"`
	Dockerfile string                    `yaml:",omitempty" json:"dockerfile,omitempty"`
	Args       compose.MappingWithEquals `yaml:",omitempty" json:"args,omitempty"`
}

func load(dict map[string]interface{}, buildArgs []string) ([]ServiceConfig, error) {
	section, ok := dict["services"]
	if !ok {
		return nil, fmt.Errorf("Compose file doesn't declare any service")
	}
	services, ok := section.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Invalid Compose file: 'services' should be a map")
	}
	return loadServices(services, buildArgs)
}

func loadServices(servicesDict map[string]interface{}, buildArgs []string) ([]ServiceConfig, error) {
	var services []ServiceConfig

	for name, serviceDef := range servicesDict {
		serviceConfig, err := loadService(name, serviceDef.(map[string]interface{}), buildArgs)
		if err != nil {
			return nil, err
		}
		services = append(services, *serviceConfig)
	}
	return services, nil
}

func loadService(name string, serviceDict map[string]interface{}, buildArgs []string) (*ServiceConfig, error) {
	serviceConfig := &ServiceConfig{Name: name}
	args := buildArgsToMap(buildArgs)

	if err := loader.Transform(serviceDict, serviceConfig, loader.Transformer{
		TypeOf: reflect.TypeOf(ImageBuildConfig{}),
		Func:   transformBuildConfig,
	}); err != nil {
		return nil, err
	}
	if serviceConfig.Build != nil {
		serviceConfig.Build.mergeArgs(args)
	}
	return serviceConfig, nil
}

func transformBuildConfig(data interface{}) (interface{}, error) {
	switch value := data.(type) {
	case string:
		return map[string]interface{}{"context": value}, nil
	case map[string]interface{}:
		return data, nil
	default:
		return data, errors.Errorf("invalid type %T for service build", value)
	}
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

func (m ImageBuildConfig) mergeArgs(mapToMerge map[string]string) {
	for key := range m.Args {
		if val, ok := mapToMerge[key]; ok {
			if val == "" {
				m.Args[key] = nil
			} else {
				m.Args[key] = &val
			}
		}
	}
}
