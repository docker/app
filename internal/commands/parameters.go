package commands

import (
	"fmt"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/docker/app/internal"
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

func withSendRegistryAuth(sendRegistryAuth bool) parameterOperation {
	return func(bndl *bundle.Bundle, params map[string]string) error {
		if _, ok := bndl.Parameters[internal.ParameterShareRegistryCredsName]; ok {
			val := "false"
			if sendRegistryAuth {
				val = "true"
			}
			params[internal.ParameterShareRegistryCredsName] = val
		}
		return nil
	}
}

func withOrchestratorParameters(orchestrator string, kubeNamespace string) parameterOperation {
	return func(bndl *bundle.Bundle, params map[string]string) error {
		if _, ok := bndl.Parameters[internal.ParameterOrchestratorName]; ok {
			params[internal.ParameterOrchestratorName] = orchestrator
		}
		if _, ok := bndl.Parameters[internal.ParameterKubernetesNamespaceName]; ok {
			params[internal.ParameterKubernetesNamespaceName] = kubeNamespace
		}
		return nil
	}
}

func mergeBundleParameters(c *claim.Claim, ops ...parameterOperation) error {
	bndl := c.Bundle
	userParams := map[string]string{}
	for _, op := range ops {
		if err := op(bndl, userParams); err != nil {
			return err
		}
	}
	convertedParams, err := matchParametersDefinition(userParams, bndl.Parameters)
	if err != nil {
		return err
	}
	c.Parameters, err = bundle.ValuesOrDefaults(convertedParams, bndl)
	return err
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
