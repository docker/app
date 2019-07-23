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
	bndl := installation.Bundle
	if installation.Parameters == nil {
		installation.Parameters = make(map[string]interface{})
	}
	userParams := map[string]string{}
	cfg := &mergeBundleConfig{
		bundle: bndl,
		params: userParams,
		stderr: os.Stderr,
	}
	for _, op := range ops {
		if err := op(cfg); err != nil {
			return err
		}
	}
	if err := matchAndMergeParametersDefinition(installation.Parameters, cfg.params, cfg.bundle, cfg.strictMode, cfg.stderr); err != nil {
		return err
	}
	var err error
	installation.Parameters, err = bundle.ValuesOrDefaults(installation.Parameters, bndl)
	return err
}

func getParameterFromBundle(name string, bndl *bundle.Bundle) (bundle.ParameterDefinition, bool) {
	if bndl.Parameters == nil {
		return bundle.ParameterDefinition{}, false
	}
	param, found := bndl.Parameters.Fields[name]
	return param, found
}

func matchAndMergeParametersDefinition(currentValues map[string]interface{}, parameterValues map[string]string, bundle *bundle.Bundle, strictMode bool, stderr io.Writer) error {
	for k, v := range parameterValues {
		param, ok := getParameterFromBundle(k, bundle)
		if !ok {
			if strictMode {
				return fmt.Errorf("parameter %q is not defined in the bundle", k)
			}
			fmt.Fprintf(stderr, "Warning: parameter %q is not defined in the bundle\n", k)
			continue
		}
		definition, ok := bundle.Definitions[param.Definition]
		if !ok {
			return fmt.Errorf("invalid bundle: definition not found for parameter %q", k)
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
