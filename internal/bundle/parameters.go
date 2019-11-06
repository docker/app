package bundle

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/store"
	"github.com/docker/app/types/parameters"
	cliopts "github.com/docker/cli/opts"
	"github.com/pkg/errors"
)

// MergeBundleConfig is the actual parameters and bundle parameters to be merged
type MergeBundleConfig struct {
	bundle *bundle.Bundle
	params map[string]string
	stderr io.Writer
}

// MergeBundleOpt is a functional option of the bundle parameter merge function
type MergeBundleOpt func(c *MergeBundleConfig) error

func WithFileParameters(parametersFiles []string) MergeBundleOpt {
	return func(c *MergeBundleConfig) error {
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

func WithCommandLineParameters(overrides []string) MergeBundleOpt {
	return func(c *MergeBundleConfig) error {
		d := cliopts.ConvertKVStringsToMap(overrides)
		for k, v := range d {
			c.params[k] = v
		}
		return nil
	}
}

func WithLabels(labels []string) MergeBundleOpt {
	return func(c *MergeBundleConfig) error {
		for _, l := range labels {
			if strings.HasPrefix(l, internal.Namespace) {
				return errors.Errorf("labels cannot start with %q", internal.Namespace)
			}
		}
		l := packager.DockerAppArgs{
			Labels: cliopts.ConvertKVStringsToMap(labels),
		}
		out, err := json.Marshal(l)
		if err != nil {
			return err
		}
		if _, ok := c.bundle.Parameters[internal.ParameterArgs]; ok {
			c.params[internal.ParameterArgs] = string(out)
		}
		return nil
	}
}

func WithSendRegistryAuth(sendRegistryAuth bool) MergeBundleOpt {
	return func(c *MergeBundleConfig) error {
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

func WithOrchestratorParameters(orchestrator string, kubeNamespace string) MergeBundleOpt {
	return func(c *MergeBundleConfig) error {
		if _, ok := c.bundle.Definitions[internal.ParameterOrchestratorName]; ok {
			c.params[internal.ParameterOrchestratorName] = orchestrator
		}
		if _, ok := c.bundle.Definitions[internal.ParameterKubernetesNamespaceName]; ok {
			c.params[internal.ParameterKubernetesNamespaceName] = kubeNamespace
		}
		return nil
	}
}

func WithErrorWriter(w io.Writer) MergeBundleOpt {
	return func(c *MergeBundleConfig) error {
		c.stderr = w
		return nil
	}
}

// MergeBundleParameters merges current, provided and bundle default parameters
func MergeBundleParameters(installation *store.Installation, ops ...MergeBundleOpt) error {
	bndl := installation.Bundle
	if installation.Parameters == nil {
		installation.Parameters = make(map[string]interface{})
	}
	userParams := map[string]string{}
	cfg := &MergeBundleConfig{
		bundle: bndl,
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
	installation.Parameters, err = bundle.ValuesOrDefaults(mergedValues, bndl)
	return err
}

func matchAndMergeParametersDefinition(currentValues map[string]interface{}, cfg *MergeBundleConfig) (map[string]interface{}, error) {
	mergedValues := make(map[string]interface{})
	for k, v := range currentValues {
		mergedValues[k] = v
	}
	for k, v := range cfg.params {
		param, ok := cfg.bundle.Parameters[k]
		if !ok {
			fmt.Fprintf(cfg.stderr, "Warning: parameter %q is not defined in the bundle\n", k)
			continue
		}
		definition, ok := cfg.bundle.Definitions[param.Definition]
		if !ok {
			return nil, fmt.Errorf("invalid bundle: definition not found for parameter %q", k)
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
