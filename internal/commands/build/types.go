package build

import (
	"fmt"
	"reflect"

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

func Load(dict map[string]interface{}) ([]ServiceConfig, error) {
	section, ok := dict["services"]
	if !ok {
		return nil, fmt.Errorf("compose file doesn't declare any service")
	}
	services, ok := section.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Invalid compose file: 'services' should be a map")
	}
	return LoadServices(services)
}

func LoadServices(servicesDict map[string]interface{}) ([]ServiceConfig, error) {
	var services []ServiceConfig

	for name, serviceDef := range servicesDict {
		serviceConfig, err := LoadService(name, serviceDef.(map[string]interface{}))
		if err != nil {
			return nil, err
		}
		services = append(services, *serviceConfig)
	}
	return services, nil
}

func LoadService(name string, serviceDict map[string]interface{}) (*ServiceConfig, error) {
	serviceConfig := &ServiceConfig{}
	if err := loader.Transform(serviceDict, serviceConfig, loader.Transformer{
		TypeOf: reflect.TypeOf(ImageBuildConfig{}),
		Func:   transformBuildConfig,
	}); err != nil {
		return nil, err
	}
	serviceConfig.Name = name
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
