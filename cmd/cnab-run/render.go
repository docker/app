package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	"github.com/pkg/errors"

	"github.com/docker/app/internal/formatter"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
)

func renderAction(instanceName string) error {
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

	formatDriver, ok := os.LookupEnv(internal.DockerRenderFormatEnvVar)
	if !ok {
		return fmt.Errorf("%q is undefined", internal.DockerRenderFormatEnvVar)
	}

	parameters := packager.ExtractCNABParametersValues(packager.ExtractCNABParameterMapping(app.Parameters()), os.Environ())

	rendered, err := render.Render(app, parameters, imageMap)
	if err != nil {
		return err
	}
	res, err := formatter.Format(rendered, formatDriver)
	if err != nil {
		return err
	}
	fmt.Print(res)

	return nil
}
