package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/store"
	"github.com/docker/app/types/parameters"
	cliopts "github.com/docker/cli/opts"
	"github.com/pkg/errors"
)

type mergeBundleConfig struct {
	bundle     *bundle.Bundle
	params     map[string]string
	strictMode bool
	stderr     io.Writer
}

type mergeBundleOpt func(c *mergeBundleConfig) error

func withFileParameters(parametersFiles []string) mergeBundleOpt {
	return func(c *mergeBundleConfig) error {
		p, err := parameters.LoadFiles(parametersFiles)
		if err != nil {
			return err
		}
		for k, v := range p.Flatten() {
			c.params[k] = v
		}
		return nil
	}
}

func withCommandLineParameters(overrides []string) mergeBundleOpt {
	return func(c *mergeBundleConfig) error {
		d := cliopts.ConvertKVStringsToMap(overrides)
		for k, v := range d {
			c.params[k] = v
		}
		return nil
	}
}

func withSendRegistryAuth(sendRegistryAuth bool) mergeBundleOpt {
	return func(c *mergeBundleConfig) error {
		if _, ok := c.bundle.Definitions[internal.ParameterShareRegistryCredsName]; ok {
			val := "false"
			if sendRegistryAuth {
				val = "true"
			}
			c.params[internal.ParameterShareRegistryCredsName] = val
		}
		return nil
	}
}

func withOrchestratorParameters(orchestrator string, kubeNamespace string) mergeBundleOpt {
	return func(c *mergeBundleConfig) error {
		if _, ok := c.bundle.Definitions[internal.ParameterOrchestratorName]; ok {
			c.params[internal.ParameterOrchestratorName] = orchestrator
		}
		if _, ok := c.bundle.Definitions[internal.ParameterKubernetesNamespaceName]; ok {
			c.params[internal.ParameterKubernetesNamespaceName] = kubeNamespace
		}
		return nil
	}
}

func withErrorWriter(w io.Writer) mergeBundleOpt {
	return func(c *mergeBundleConfig) error {
		c.stderr = w
		return nil
	}
}

func withStrictMode(strict bool) mergeBundleOpt {
	return func(c *mergeBundleConfig) error {
		c.strictMode = strict
		return nil
	}
}

func mergeBundleParameters(installation *store.Installation, ops ...mergeBundleOpt) error {
	if installation.Parameters == nil {
		installation.Parameters = make(map[string]interface{})
	}
	userParams := map[string]string{}
	cfg := &mergeBundleConfig{
		bundle: installation.Bundle,
		params: userParams,
		stderr: os.Stderr,
	}
	for _, op := range ops {
		if err := op(cfg); err != nil {
			return err
		}
	}
	mergedValues, err := matchAndMergeParametersDefinition(installation.Parameters, cfg)
	if err != nil {
		return err
	}
	installation.Parameters, err = bundle.ValuesOrDefaults(mergedValues, installation.Parameters, installation.Bundle)
	return err
}

func matchAndMergeParametersDefinition(currentValues map[string]interface{}, cfg *mergeBundleConfig) (map[string]interface{}, error) {
	mergedValues := make(map[string]interface{})
	for k, v := range currentValues {
		mergedValues[k] = v
	}
	for k, v := range cfg.params {
		definition, ok := cfg.bundle.Definitions[k]
		if !ok {
			if cfg.strictMode {
				return nil, fmt.Errorf("parameter %q is not defined in the bundle", k)
			}
			fmt.Fprintf(cfg.stderr, "Warning: parameter %q is not defined in the bundle\n", k)
			continue
		}
		value, err := definition.ConvertValue(v)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid value for parameter %q", k)
		}
		valErrors, err := definition.Validate(value)
		if valErrors != nil {
			errs := make([]string, len(valErrors))
			for i, v := range valErrors {
				errs[i] = v.Error
			}
			errMsg := strings.Join(errs, ", ")
			return nil, errors.Wrapf(fmt.Errorf(errMsg), "invalid value for parameter %q", k)
		}
		if err != nil {
			return nil, errors.Wrapf(err, "invalid value for parameter %q", k)
		}
		mergedValues[k] = value
	}
	return mergedValues, nil
}
