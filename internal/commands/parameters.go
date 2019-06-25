package commands

import (
	"fmt"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/bundle/definition"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/store"
	"github.com/docker/app/types/parameters"
	cliopts "github.com/docker/cli/opts"
	"github.com/pkg/errors"
)

type mergeBundleOpt func(bndl *bundle.Bundle, params map[string]string) error

func withFileParameters(parametersFiles []string) mergeBundleOpt {
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

func withCommandLineParameters(overrides []string) mergeBundleOpt {
	return func(bndl *bundle.Bundle, params map[string]string) error {
		d := cliopts.ConvertKVStringsToMap(overrides)
		for k, v := range d {
			params[k] = v
		}
		return nil
	}
}

func withSendRegistryAuth(sendRegistryAuth bool) mergeBundleOpt {
	return func(bndl *bundle.Bundle, params map[string]string) error {
		if _, ok := bndl.Definitions[internal.ParameterShareRegistryCredsName]; ok {
			val := "false"
			if sendRegistryAuth {
				val = "true"
			}
			params[internal.ParameterShareRegistryCredsName] = val
		}
		return nil
	}
}

func withOrchestratorParameters(orchestrator string, kubeNamespace string) mergeBundleOpt {
	return func(bndl *bundle.Bundle, params map[string]string) error {
		if _, ok := bndl.Definitions[internal.ParameterOrchestratorName]; ok {
			params[internal.ParameterOrchestratorName] = orchestrator
		}
		if _, ok := bndl.Definitions[internal.ParameterKubernetesNamespaceName]; ok {
			params[internal.ParameterKubernetesNamespaceName] = kubeNamespace
		}
		return nil
	}
}

func mergeBundleParameters(installation *store.Installation, ops ...mergeBundleOpt) error {
	bndl := installation.Bundle
	if installation.Parameters == nil {
		installation.Parameters = make(map[string]interface{})
	}
	userParams := map[string]string{}
	for _, op := range ops {
		if err := op(bndl, userParams); err != nil {
			return err
		}
	}
	if err := matchAndMergeParametersDefinition(installation.Parameters, userParams, bndl.Definitions); err != nil {
		return err
	}
	var err error
	installation.Parameters, err = bundle.ValuesOrDefaults(installation.Parameters, bndl)
	return err
}

func matchAndMergeParametersDefinition(currentValues map[string]interface{}, parameterValues map[string]string, parameterDefinitions definition.Definitions) error {
	for k, v := range parameterValues {
		definition, ok := parameterDefinitions[k]
		if !ok {
			return fmt.Errorf("parameter %q is not defined in the bundle", k)
		}
		value, err := definition.ConvertValue(v)
		if err != nil {
			return errors.Wrapf(err, "invalid value for parameter %q", k)
		}
		valErrors, err := definition.Validate(value)
		if valErrors != nil {
			errs := make([]string, len(valErrors))
			for i, v := range valErrors {
				errs[i] = v.Error
			}
			errMsg := strings.Join(errs, ", ")
			return errors.Wrapf(fmt.Errorf(errMsg), "invalid value for parameter %q", k)
		}
		if err != nil {
			return errors.Wrapf(err, "invalid value for parameter %q", k)
		}
		currentValues[k] = value
	}
	return nil
}
