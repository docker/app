package main

import (
	"fmt"
	"os"

	"github.com/docker/app/internal"

	"github.com/docker/app/internal/formatter"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
)

func renderAction(instanceName string) error {
	app, err := packager.Extract("")
	// todo: merge additional compose file
	if err != nil {
		return err
	}
	defer app.Cleanup()

	bndl, err := getBundle()
	if err != nil {
		return err
	}

	formatDriver, ok := os.LookupEnv(internal.DockerRenderFormatEnvVar)
	if !ok {
		return fmt.Errorf("%q is undefined", internal.DockerRenderFormatEnvVar)
	}

	parameters := packager.ExtractCNABParametersValues(packager.ExtractCNABParameterMapping(app.Parameters()), os.Environ())

	rendered, err := render.Render(app, parameters, bndl.Images)
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
