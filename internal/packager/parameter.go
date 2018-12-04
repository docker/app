package packager

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docker/app/types/parameters"
)

// CNABParametersMapping describes the desired mapping between parameters and CNAB environment variables
type CNABParametersMapping struct {
	CNABEnvToParameter map[string]string
	ParameterToCNABEnv map[string]string
}

// ExtractCNABParameterMapping extracts the CNABParametersMapping from application parameters
func ExtractCNABParameterMapping(parameters parameters.Parameters) CNABParametersMapping {
	keys := getKeys("", parameters)
	sort.Strings(keys)
	mapping := CNABParametersMapping{
		CNABEnvToParameter: make(map[string]string),
		ParameterToCNABEnv: make(map[string]string),
	}
	for ix, key := range keys {
		env := fmt.Sprintf("docker_param%d", ix+1)
		mapping.CNABEnvToParameter[env] = key
		mapping.ParameterToCNABEnv[key] = env
	}
	return mapping
}

func getKeys(prefix string, parameters map[string]interface{}) []string {
	var keys []string
	for k, v := range parameters {
		sub, ok := v.(map[string]interface{})
		if ok {
			subPrefix := prefix
			subPrefix += fmt.Sprintf("%s.", k)
			keys = append(keys, getKeys(subPrefix, sub)...)
		} else {
			keys = append(keys, prefix+k)
		}
	}
	return keys
}

// ExtractCNABParametersValues extracts the parameter values from the given CNAB environment
func ExtractCNABParametersValues(mapping CNABParametersMapping, env []string) map[string]string {
	envValues := map[string]string{}
	for _, v := range env {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			if key, ok := mapping.CNABEnvToParameter[parts[0]]; ok {
				envValues[key] = parts[1]
			}
		}
	}
	return envValues
}
