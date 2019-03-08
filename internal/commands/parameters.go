package commands

import (
	"fmt"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/app/types/parameters"
	cliopts "github.com/docker/cli/opts"
	"github.com/pkg/errors"
)

type parameterOperation func(bndl *bundle.Bundle, params map[string]string) error

func withFileParameters(parametersFiles []string) parameterOperation {
	return func(bndl *bundle.Bundle, params map[string]string) error {
		p, err := parameters.LoadFiles(parametersFiles)
		if err != nil {
			return err
		}
		for k, v := range p.Flatten() {
			params[k] = v
		}
		return nil
	}
}

func withCommandLineParameters(overrides []string) parameterOperation {
	return func(bndl *bundle.Bundle, params map[string]string) error {
		d := cliopts.ConvertKVStringsToMap(overrides)
		for k, v := range d {
			params[k] = v
		}
		return nil
	}
}

func withOrchestratorParameters(orchestrator string, kubeNamespace string) parameterOperation {
	return func(bndl *bundle.Bundle, params map[string]string) error {
		if _, ok := bndl.Parameters["docker.orchestrator"]; ok {
			params["docker.orchestrator"] = orchestrator
		}
		if _, ok := bndl.Parameters["docker.kubernetes-namespace"]; ok {
			params["docker.kubernetes-namespace"] = kubeNamespace
		}
		return nil
	}
}

func mergeBundleParameters(bndl *bundle.Bundle, ops ...parameterOperation) (map[string]interface{}, error) {
	userParams := map[string]string{}
	for _, op := range ops {
		if err := op(bndl, userParams); err != nil {
			return nil, err
		}
	}
	convertedParams, err := matchParametersDefinition(userParams, bndl.Parameters)
	if err != nil {
		return nil, err
	}
	return bundle.ValuesOrDefaults(convertedParams, bndl)
}

func matchParametersDefinition(parameterValues map[string]string, parameterDefinitions map[string]bundle.ParameterDefinition) (map[string]interface{}, error) {
	finalValues := map[string]interface{}{}
	for k, v := range parameterValues {
		definition, ok := parameterDefinitions[k]
		if !ok {
			return nil, fmt.Errorf("parameter %q is not defined in the bundle", k)
		}
		value, err := definition.ConvertValue(v)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid value for parameter %q", k)
		}
		if err := definition.ValidateParameterValue(value); err != nil {
			return nil, errors.Wrapf(err, "invalid value for parameter %q", k)
		}
		finalValues[k] = value
	}
	return finalValues, nil
}
