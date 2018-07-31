package templateloader

import (
	"fmt"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/app/internal/helm/templatetypes"
	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/schema"
	"github.com/docker/cli/cli/compose/template"
	"github.com/docker/cli/cli/compose/types"
	"github.com/docker/cli/opts"
	"github.com/docker/go-connections/nat"
	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	transformers = []loader.Transformer{
		{TypeOf: reflect.TypeOf(templatetypes.UnitBytesOrTemplate{}), Func: transformSize},
		{TypeOf: reflect.TypeOf([]templatetypes.ServicePortConfig{}), Func: transformServicePort},
		{TypeOf: reflect.TypeOf(templatetypes.ServiceSecretConfig{}), Func: transformStringSourceMap},
		{TypeOf: reflect.TypeOf(templatetypes.ServiceConfigObjConfig{}), Func: transformStringSourceMap},
		{TypeOf: reflect.TypeOf(templatetypes.ServiceVolumeConfig{}), Func: transformServiceVolumeConfig},
		{TypeOf: reflect.TypeOf(templatetypes.BoolOrTemplate{}), Func: transformBoolOrTemplate},
		{TypeOf: reflect.TypeOf(templatetypes.UInt64OrTemplate{}), Func: transformUInt64OrTemplate},
		{TypeOf: reflect.TypeOf(templatetypes.DurationOrTemplate{}), Func: transformDurationOrTemplate},
	}
)

// LoadTemplate loads a config without resolving the variables
func LoadTemplate(configDict map[string]interface{}) (*templatetypes.Config, error) {
	if err := validateForbidden(configDict); err != nil {
		return nil, err
	}
	return loadSections(configDict, types.ConfigDetails{})
}

func validateForbidden(configDict map[string]interface{}) error {
	servicesDict, ok := configDict["services"].(map[string]interface{})
	if !ok {
		return nil
	}
	forbidden := getProperties(servicesDict, types.ForbiddenProperties)
	if len(forbidden) > 0 {
		return &ForbiddenPropertiesError{Properties: forbidden}
	}
	return nil
}

func loadSections(config map[string]interface{}, configDetails types.ConfigDetails) (*templatetypes.Config, error) {
	var err error
	cfg := templatetypes.Config{
		Version: schema.Version(config),
	}

	var loaders = []struct {
		key string
		fnc func(config map[string]interface{}) error
	}{
		{
			key: "services",
			fnc: func(config map[string]interface{}) error {
				cfg.Services, err = LoadServices(config, configDetails.WorkingDir, configDetails.LookupEnv)
				return err
			},
		},
		{
			key: "networks",
			fnc: func(config map[string]interface{}) error {
				cfg.Networks, err = loader.LoadNetworks(config, configDetails.Version)
				return err
			},
		},
		{
			key: "volumes",
			fnc: func(config map[string]interface{}) error {
				cfg.Volumes, err = loader.LoadVolumes(config, configDetails.Version)
				return err
			},
		},
		{
			key: "secrets",
			fnc: func(config map[string]interface{}) error {
				cfg.Secrets, err = loader.LoadSecrets(config, configDetails)
				return err
			},
		},
		{
			key: "configs",
			fnc: func(config map[string]interface{}) error {
				cfg.Configs, err = loader.LoadConfigObjs(config, configDetails)
				return err
			},
		},
	}
	for _, loader := range loaders {
		if err := loader.fnc(getSection(config, loader.key)); err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

func getSection(config map[string]interface{}, key string) map[string]interface{} {
	section, ok := config[key]
	if !ok {
		return make(map[string]interface{})
	}
	return section.(map[string]interface{})
}

// GetUnsupportedProperties returns the list of any unsupported properties that are
// used in the Compose files.
func GetUnsupportedProperties(configDicts ...map[string]interface{}) []string {
	unsupported := map[string]bool{}

	for _, configDict := range configDicts {
		for _, service := range getServices(configDict) {
			serviceDict := service.(map[string]interface{})
			for _, property := range types.UnsupportedProperties {
				if _, isSet := serviceDict[property]; isSet {
					unsupported[property] = true
				}
			}
		}
	}

	return sortedKeys(unsupported)
}

func sortedKeys(set map[string]bool) []string {
	var keys []string
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// GetDeprecatedProperties returns the list of any deprecated properties that
// are used in the compose files.
func GetDeprecatedProperties(configDicts ...map[string]interface{}) map[string]string {
	deprecated := map[string]string{}

	for _, configDict := range configDicts {
		deprecatedProperties := getProperties(getServices(configDict), types.DeprecatedProperties)
		for key, value := range deprecatedProperties {
			deprecated[key] = value
		}
	}

	return deprecated
}

func getProperties(services map[string]interface{}, propertyMap map[string]string) map[string]string {
	output := map[string]string{}

	for _, service := range services {
		if serviceDict, ok := service.(map[string]interface{}); ok {
			for property, description := range propertyMap {
				if _, isSet := serviceDict[property]; isSet {
					output[property] = description
				}
			}
		}
	}

	return output
}

// ForbiddenPropertiesError is returned when there are properties in the Compose
// file that are forbidden.
type ForbiddenPropertiesError struct {
	Properties map[string]string
}

func (e *ForbiddenPropertiesError) Error() string {
	return "Configuration contains forbidden properties"
}

func getServices(configDict map[string]interface{}) map[string]interface{} {
	if services, ok := configDict["services"]; ok {
		if servicesDict, ok := services.(map[string]interface{}); ok {
			return servicesDict
		}
	}

	return map[string]interface{}{}
}

// LoadServices produces a ServiceConfig map from a compose file Dict
// the servicesDict is not validated if directly used. Use Load() to enable validation
func LoadServices(servicesDict map[string]interface{}, workingDir string, lookupEnv template.Mapping) ([]templatetypes.ServiceConfig, error) {
	var services []templatetypes.ServiceConfig

	for name, serviceDef := range servicesDict {
		serviceConfig, err := LoadService(name, serviceDef.(map[string]interface{}), workingDir, lookupEnv)
		if err != nil {
			return nil, err
		}
		services = append(services, *serviceConfig)
	}

	return services, nil
}

// LoadService produces a single ServiceConfig from a compose file Dict
// the serviceDict is not validated if directly used. Use Load() to enable validation
func LoadService(name string, serviceDict map[string]interface{}, workingDir string, lookupEnv template.Mapping) (*templatetypes.ServiceConfig, error) {
	serviceConfig := &templatetypes.ServiceConfig{}
	if err := loader.Transform(serviceDict, serviceConfig, transformers...); err != nil {
		return nil, err
	}
	serviceConfig.Name = name

	if err := resolveEnvironment(serviceConfig, workingDir, lookupEnv); err != nil {
		return nil, err
	}

	if err := resolveVolumePaths(serviceConfig.Volumes, workingDir, lookupEnv); err != nil {
		return nil, err
	}
	return serviceConfig, nil
}

func updateEnvironment(environment map[string]*string, vars map[string]*string, lookupEnv template.Mapping) {
	for k, v := range vars {
		interpolatedV, ok := lookupEnv(k)
		if (v == nil || *v == "") && ok {
			// lookupEnv is prioritized over vars
			environment[k] = &interpolatedV
		} else {
			environment[k] = v
		}
	}
}

func resolveEnvironment(serviceConfig *templatetypes.ServiceConfig, workingDir string, lookupEnv template.Mapping) error {
	environment := make(map[string]*string)

	if len(serviceConfig.EnvFile) > 0 {
		var envVars []string

		for _, file := range serviceConfig.EnvFile {
			filePath := absPath(workingDir, file)
			fileVars, err := opts.ParseEnvFile(filePath)
			if err != nil {
				return err
			}
			envVars = append(envVars, fileVars...)
		}
		updateEnvironment(environment,
			opts.ConvertKVStringsToMapWithNil(envVars), lookupEnv)
	}

	updateEnvironment(environment, serviceConfig.Environment, lookupEnv)
	serviceConfig.Environment = environment
	return nil
}

func resolveVolumePaths(volumes []templatetypes.ServiceVolumeConfig, workingDir string, lookupEnv template.Mapping) error {
	for i, volume := range volumes {
		if volume.Type != "bind" {
			continue
		}

		if volume.Source == "" {
			return errors.New(`invalid mount config for type "bind": field Source must not be empty`)
		}

		filePath := expandUser(volume.Source, lookupEnv)
		// Check for a Unix absolute path first, to handle a Windows client
		// with a Unix daemon. This handles a Windows client connecting to a
		// Unix daemon. Note that this is not required for Docker for Windows
		// when specifying a local Windows path, because Docker for Windows
		// translates the Windows path into a valid path within the VM.
		if !path.IsAbs(filePath) {
			filePath = absPath(workingDir, filePath)
		}
		volume.Source = filePath
		volumes[i] = volume
	}
	return nil
}

// TODO: make this more robust
func expandUser(path string, lookupEnv template.Mapping) string {
	if strings.HasPrefix(path, "~") {
		home, ok := lookupEnv("HOME")
		if !ok {
			logrus.Warn("cannot expand '~', because the environment lacks HOME")
			return path
		}
		return strings.Replace(path, "~", home, 1)
	}
	return path
}

func absPath(workingDir string, filePath string) string {
	if filepath.IsAbs(filePath) {
		return filePath
	}
	return filepath.Join(workingDir, filePath)
}

func transformServicePort(data interface{}) (interface{}, error) {
	switch entries := data.(type) {
	case []interface{}:
		// We process the list instead of individual items here.
		// The reason is that one entry might be mapped to multiple ServicePortConfig.
		// Therefore we take an input of a list and return an output of a list.
		ports := []interface{}{}
		for _, entry := range entries {
			switch value := entry.(type) {
			case int:
				v, err := toServicePortConfigs(fmt.Sprint(value))
				if err != nil {
					return data, err
				}
				ports = append(ports, v...)
			case string:
				v, err := toServicePortConfigs(value)
				if err != nil {
					return data, err
				}
				ports = append(ports, v...)
			case map[string]interface{}:
				ports = append(ports, value)
			default:
				return data, errors.Errorf("invalid type %T for port", value)
			}
		}
		return ports, nil
	default:
		return data, errors.Errorf("invalid type %T for port", entries)
	}
}

func transformStringSourceMap(data interface{}) (interface{}, error) {
	switch value := data.(type) {
	case string:
		return map[string]interface{}{"source": value}, nil
	case map[string]interface{}:
		return data, nil
	default:
		return data, errors.Errorf("invalid type %T for secret", value)
	}
}

func transformServiceVolumeConfig(data interface{}) (interface{}, error) {
	switch value := data.(type) {
	case string:
		return ParseVolume(value)
	case map[string]interface{}:
		return data, nil
	default:
		return data, errors.Errorf("invalid type %T for service volume", value)
	}
}

func transformBoolOrTemplate(value interface{}) (interface{}, error) {
	switch value := value.(type) {
	case int:
		return templatetypes.BoolOrTemplate{Value: value != 0}, nil
	case bool:
		return templatetypes.BoolOrTemplate{Value: value}, nil
	case string:
		b, err := toBoolean(value)
		if err == nil {
			return templatetypes.BoolOrTemplate{Value: b.(bool)}, nil
		}
		return templatetypes.BoolOrTemplate{ValueTemplate: value}, nil
	default:
		return value, errors.Errorf("invali type %T for boolean", value)
	}
}

func transformUInt64OrTemplate(value interface{}) (interface{}, error) {
	switch value := value.(type) {
	case int:
		v := uint64(value)
		return templatetypes.UInt64OrTemplate{Value: &v}, nil
	case string:
		v, err := strconv.ParseUint(value, 0, 64)
		if err == nil {
			return templatetypes.UInt64OrTemplate{Value: &v}, nil
		}
		return templatetypes.UInt64OrTemplate{ValueTemplate: value}, nil
	default:
		return value, errors.Errorf("invali type %T for boolean", value)
	}
}

func transformDurationOrTemplate(value interface{}) (interface{}, error) {
	switch value := value.(type) {
	case int:
		d := time.Duration(value)
		return templatetypes.DurationOrTemplate{Value: &d}, nil
	case string:
		d, err := time.ParseDuration(value)
		if err == nil {
			return templatetypes.DurationOrTemplate{Value: &d}, nil
		}
		return templatetypes.DurationOrTemplate{ValueTemplate: value}, nil
	default:
		return nil, errors.Errorf("invalid type for duration %T", value)
	}
}

func transformSize(value interface{}) (interface{}, error) {
	switch value := value.(type) {
	case int:
		return templatetypes.UnitBytesOrTemplate{Value: int64(value)}, nil
	case string:
		v, err := units.RAMInBytes(value)
		if err == nil {
			return templatetypes.UnitBytesOrTemplate{Value: int64(v)}, nil
		}
		return templatetypes.UnitBytesOrTemplate{ValueTemplate: value}, nil
	}
	return nil, errors.Errorf("invalid type for size %T", value)
}

func toServicePortConfigs(value string) ([]interface{}, error) {
	var portConfigs []interface{}
	if strings.Contains(value, "$") {
		// template detected
		if strings.Contains(value, "-") {
			return nil, fmt.Errorf("port range not supported with templated values")
		}
		portsProtocol := strings.Split(value, "/")
		protocol := "tcp"
		if len(portsProtocol) > 1 {
			protocol = portsProtocol[1]
		}
		portPort := strings.Split(portsProtocol[0], ":")
		tgt, _ := transformUInt64OrTemplate(portPort[0]) // can't fail on string
		pub := templatetypes.UInt64OrTemplate{}
		if len(portPort) > 1 {
			ipub, _ := transformUInt64OrTemplate(portPort[1])
			pub = ipub.(templatetypes.UInt64OrTemplate)
		}
		portConfigs = append(portConfigs, templatetypes.ServicePortConfig{
			Protocol:  protocol,
			Target:    tgt.(templatetypes.UInt64OrTemplate),
			Published: pub,
			Mode:      "ingress",
		})
		return portConfigs, nil
	}

	ports, portBindings, err := nat.ParsePortSpecs([]string{value})
	if err != nil {
		return nil, err
	}
	// We need to sort the key of the ports to make sure it is consistent
	keys := []string{}
	for port := range ports {
		keys = append(keys, string(port))
	}
	sort.Strings(keys)

	for _, key := range keys {
		// Reuse ConvertPortToPortConfig so that it is consistent
		portConfig, err := opts.ConvertPortToPortConfig(nat.Port(key), portBindings)
		if err != nil {
			return nil, err
		}
		for _, p := range portConfig {
			tp := uint64(p.TargetPort)
			pp := uint64(p.PublishedPort)
			portConfigs = append(portConfigs, templatetypes.ServicePortConfig{
				Protocol:  string(p.Protocol),
				Target:    templatetypes.UInt64OrTemplate{Value: &tp},
				Published: templatetypes.UInt64OrTemplate{Value: &pp},
				Mode:      string(p.PublishMode),
			})
		}
	}

	return portConfigs, nil
}
