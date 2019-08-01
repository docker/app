package action

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
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
	Run(*claim.Claim, credentials.Set, io.Writer) error
}

func selectInvocationImage(d driver.Driver, c *claim.Claim) (bundle.InvocationImage, error) {
	if len(c.Bundle.InvocationImages) == 0 {
		return bundle.InvocationImage{}, errors.New("no invocationImages are defined in the bundle")
	}

	for _, ii := range c.Bundle.InvocationImages {
		if d.Handles(ii.ImageType) {
			if c.RelocationMap != nil {
				if img, ok := c.RelocationMap[ii.Image]; ok {
					ii.Image = img
				}
			}
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

func appliesToAction(action string, parameter bundle.Parameter) bool {
	if len(parameter.ApplyTo) == 0 {
		return true
	}
	for _, act := range parameter.ApplyTo {
		if action == act {
			return true
		}
	}
	return false
}

func opFromClaim(action string, stateless bool, c *claim.Claim, ii bundle.InvocationImage, creds credentials.Set, w io.Writer) (*driver.Operation, error) {
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

	imgMap, err := getImageMap(c.Bundle)
	if err != nil {
		return nil, fmt.Errorf("unable to generate image map: %s", err)
	}
	files["/cnab/app/image-map.json"] = string(imgMap)

	env["CNAB_INSTALLATION_NAME"] = c.Name
	env["CNAB_ACTION"] = action
	env["CNAB_BUNDLE_NAME"] = c.Bundle.Name
	env["CNAB_BUNDLE_VERSION"] = c.Bundle.Version

	return &driver.Operation{
		Action:       action,
		Installation: c.Name,
		Parameters:   c.Parameters,
		Image:        ii.Image,
		ImageType:    ii.ImageType,
		Revision:     c.Revision,
		Environment:  env,
		Files:        files,
		Out:          w,
	}, nil
}

func injectParameters(action string, c *claim.Claim, env, files map[string]string) error {
	for k, param := range c.Bundle.Parameters {
		rawval, ok := c.Parameters[k]
		if !ok {
			if param.Required && appliesToAction(action, param) {
				return fmt.Errorf("missing required parameter %q for action %q", k, action)
			}
			continue
		}
		value := fmt.Sprintf("%v", rawval)
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
