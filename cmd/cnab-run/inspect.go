package main

import (
	"bytes"
	"os"

	appinspect "github.com/docker/app/internal/inspect"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/types"
	"github.com/pkg/errors"
)

func inspectAction(instanceName string) error {
	overrides, err := parseOverrides()
	if err != nil {
		return errors.Wrap(err, "unable to parse auto-parameter values")
	}
	app, err := packager.Extract("", types.WithComposes(bytes.NewReader(overrides)))
	// todo: merge additional compose file
	if err != nil {
		return err
	}
	defer app.Cleanup()

	imageMap, err := getBundleImageMap()
	if err != nil {
		return err
	}

	parameters := packager.ExtractCNABParametersValues(packager.ExtractCNABParameterMapping(app.Parameters()), os.Environ())
	return appinspect.Inspect(os.Stdout, app, parameters, imageMap)
}
