package main

import (
	"fmt"
	"os"
	"time"

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

	imageMap, err := getBundleImageMap()
	if err != nil {
		return err
	}

	formatDriver, ok := os.LookupEnv(internal.DockerRenderFormatEnvVar)
	if !ok {
		return fmt.Errorf("%q is undefined", internal.DockerRenderFormatEnvVar)
	}

	parameters := packager.ExtractCNABParametersValues(packager.ExtractCNABParameterMapping(app.Parameters()), os.Environ())

	fmt.Fprintf(os.Stderr, "%s: cnab-run render on instance %q with formatter %q\n", time.Now(), instanceName, formatDriver)

	rendered, err := render.Render(app, parameters, imageMap)
	if err != nil {
		return err
	}
	res, err := formatter.Format(rendered, formatDriver)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "%s: BEGIN FORMATTED OUTPUT\n", time.Now())
	fmt.Fprintf(os.Stderr, "%s\n", res)
	fmt.Fprintf(os.Stderr, "%s: END FORMATTED OUTPUT\n", time.Now())
	fmt.Print(res)

	return nil
}
