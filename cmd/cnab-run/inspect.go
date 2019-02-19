package main

import (
	"os"

	appinspect "github.com/docker/app/internal/inspect"
	"github.com/docker/app/internal/packager"
)

func inspect() error {
	app, err := packager.Extract("")
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
