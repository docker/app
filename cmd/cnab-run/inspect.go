package main

import (
	"os"

	appinspect "github.com/docker/app/internal/inspect"
	"github.com/docker/app/internal/packager"
)

func inspectAction(instanceName string) error {
	app, err := packager.Extract("")
	// todo: merge additional compose file
	if err != nil {
		return err
	}
	defer app.Cleanup()

	bndl, err := getRelocatedBundle()
	if err != nil {
		return err
	}

	parameters := packager.ExtractCNABParametersValues(packager.ExtractCNABParameterMapping(app.Parameters()), os.Environ())
	return appinspect.ImageInspect(os.Stdout, app, parameters, bndl.RelocatedImages())
}
