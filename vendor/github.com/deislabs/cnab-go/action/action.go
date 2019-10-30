package action

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/bundle/definition"
	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/deislabs/cnab-go/driver"
)

// stateful is there just to make callers of opFromClaims more readable
const stateful = false

// Action describes one of the primary actions that can be executed in CNAB.
//
// The actions are:
// - install
// - upgrade
// - uninstall
// - downgrade
// - status
type Action interface {
	// Run an action, and record the status in the given claim
	Run(*claim.Claim, credentials.Set, ...OperationConfigFunc) error
}

func golangTypeToJSONType(value interface{}) (string, error) {
	switch v := value.(type) {
	case nil:
		return "null", nil
	case bool:
		return "boolean", nil
	case float64:
		// All numeric values are parsed by JSON into float64s. When a value could be an integer, it could also be a number, so give the more specific answer.
		if math.Trunc(v) == v {
			return "integer", nil
		}
		return "number", nil
	case string:
		return "string", nil
	case map[string]interface{}:
		return "object", nil
	case []interface{}:
		return "array", nil
	default:
		return fmt.Sprintf("%T", value), fmt.Errorf("unsupported type: %T", value)
	}
}

// allowedTypes takes an output Schema and returns a map of the allowed types (to true)
// or an error (if reading the allowed types from the schema failed).
func allowedTypes(outputSchema definition.Schema) (map[string]bool, error) {
	var outputTypes []string
	mapOutputTypes := map[string]bool{}

	// Get list of one or more allowed types for this output
	outputType, ok, err1 := outputSchema.GetType()
	if !ok { // there are multiple types
		var err2 error
		outputTypes, ok, err2 = outputSchema.GetTypes()
		if !ok {
			return mapOutputTypes, fmt.Errorf("Getting a single type errored with %q and getting multiple types errored with %q", err1, err2)
		}
	} else {
		outputTypes = []string{outputType}
	}

	// Turn allowed outputs into map for easier membership checking
	for _, v := range outputTypes {
		mapOutputTypes[v] = true
	}

	// All integers make acceptable numbers, and our helper function provides the most specific type.
	if mapOutputTypes["number"] {
		mapOutputTypes["integer"] = true
	}

	return mapOutputTypes, nil
}

// keys takes a map and returns the keys joined into a comma-separate string.
func keys(stringMap map[string]bool) string {
	var keys []string
	for k := range stringMap {
		keys = append(keys, k)
	}
	return strings.Join(keys, ",")
}

// isTypeOK uses the content and allowedTypes arguments to make sure the content of an output file matches one of the allowed types.
// The other arguments (name and allowedTypesList) are used when assembling error messages.
func isTypeOk(name, content string, allowedTypes map[string]bool) error {
	if !allowedTypes["string"] { // String output types are always passed through as the escape hatch for non-JSON bundle outputs.
		var value interface{}
		if err := json.Unmarshal([]byte(content), &value); err != nil {
			return fmt.Errorf("failed to parse %q: %s", name, err)
		}

		v, err := golangTypeToJSONType(value)
		if err != nil {
			return fmt.Errorf("%q is not a known JSON type. Expected %q to be one of: %s", name, v, keys(allowedTypes))
		}
		if !allowedTypes[v] {
			return fmt.Errorf("%q is not any of the expected types (%s) because it is %q", name, keys(allowedTypes), v)
		}
	}
	return nil
}

func setOutputsOnClaim(claim *claim.Claim, outputs map[string]string) error {
	var outputErrors []error
	claim.Outputs = map[string]interface{}{}

	if claim.Bundle.Outputs == nil {
		return nil
	}

	for outputName, v := range claim.Bundle.Outputs {
		name := v.Definition
		if name == "" {
			return fmt.Errorf("invalid bundle: no definition set for output %q", outputName)
		}

		outputSchema := claim.Bundle.Definitions[name]
		if outputSchema == nil {
			return fmt.Errorf("invalid bundle: output %q references definition %q, which was not found", outputName, name)
		}
		outputTypes, err := allowedTypes(*outputSchema)
		if err != nil {
			return err
		}

		content := outputs[v.Path]
		if content != "" {
			err := isTypeOk(outputName, content, outputTypes)
			if err != nil {
				outputErrors = append(outputErrors, err)
			}
			claim.Outputs[outputName] = outputs[v.Path]
		}
	}

	if len(outputErrors) > 0 {
		return fmt.Errorf("error: %s", outputErrors)
	}

	return nil
}

func selectInvocationImage(d driver.Driver, c *claim.Claim) (bundle.InvocationImage, error) {
	if len(c.Bundle.InvocationImages) == 0 {
		return bundle.InvocationImage{}, errors.New("no invocationImages are defined in the bundle")
	}

	for _, ii := range c.Bundle.InvocationImages {
		if d.Handles(ii.ImageType) {
			return ii, nil
		}
	}

	return bundle.InvocationImage{}, errors.New("driver is not compatible with any of the invocation images in the bundle")
}

func getImageMap(b *bundle.Bundle) ([]byte, error) {
	imgs := b.Images
	if imgs == nil {
		imgs = make(map[string]bundle.Image)
	}
	return json.Marshal(imgs)
}

func opFromClaim(action string, stateless bool, c *claim.Claim, ii bundle.InvocationImage, creds credentials.Set) (*driver.Operation, error) {
	env, files, err := creds.Expand(c.Bundle, stateless)
	if err != nil {
		return nil, err
	}

	// Quick verification that no params were passed that are not actual legit params.
	for key := range c.Parameters {
		if _, ok := c.Bundle.Parameters[key]; !ok {
			return nil, fmt.Errorf("undefined parameter %q", key)
		}
	}

	if err := injectParameters(action, c, env, files); err != nil {
		return nil, err
	}

	bundleBytes, err := json.Marshal(c.Bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bundle contents: %s", err)
	}

	files["/cnab/bundle.json"] = string(bundleBytes)

	imgMap, err := getImageMap(c.Bundle)
	if err != nil {
		return nil, fmt.Errorf("unable to generate image map: %s", err)
	}
	files["/cnab/app/image-map.json"] = string(imgMap)

	env["CNAB_INSTALLATION_NAME"] = c.Name
	env["CNAB_ACTION"] = action
	env["CNAB_BUNDLE_NAME"] = c.Bundle.Name
	env["CNAB_BUNDLE_VERSION"] = c.Bundle.Version

	var outputs []string
	if c.Bundle.Outputs != nil {
		for _, v := range c.Bundle.Outputs {
			outputs = append(outputs, v.Path)
		}
	}

	return &driver.Operation{
		Action:       action,
		Installation: c.Name,
		Parameters:   c.Parameters,
		Image:        ii,
		Revision:     c.Revision,
		Environment:  env,
		Files:        files,
		Outputs:      outputs,
		Bundle:       c.Bundle,
	}, nil
}

func injectParameters(action string, c *claim.Claim, env, files map[string]string) error {
	for k, param := range c.Bundle.Parameters {
		rawval, ok := c.Parameters[k]
		if !ok {
			if param.Required && param.AppliesTo(action) {
				return fmt.Errorf("missing required parameter %q for action %q", k, action)
			}
			continue
		}

		contents, err := json.Marshal(rawval)
		if err != nil {
			return err
		}

		// In order to preserve the exact string value the user provided
		// we don't marshal string parameters
		value := string(contents)
		if value[0] == '"' {
			value, ok = rawval.(string)
			if !ok {
				return fmt.Errorf("failed to parse parameter %q as string", k)
			}
		}

		if param.Destination == nil {
			// env is a CNAB_P_
			env[fmt.Sprintf("CNAB_P_%s", strings.ToUpper(k))] = value
			continue
		}
		if param.Destination.Path != "" {
			files[param.Destination.Path] = value
		}
		if param.Destination.EnvironmentVariable != "" {
			env[param.Destination.EnvironmentVariable] = value
		}
	}
	return nil
}
