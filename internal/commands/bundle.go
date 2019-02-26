package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/types"
	"github.com/docker/app/types/metadata"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type bundleOptions struct {
	out string
}

func bundleCmd(dockerCli command.Cli) *cobra.Command {
	var opts bundleOptions
	cmd := &cobra.Command{
		Use:   "bundle [<app-name>]",
		Short: "Create a CNAB invocation image and bundle.json for the application.",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBundle(dockerCli, firstOrEmpty(args), opts)
		},
	}

	cmd.Flags().StringVarP(&opts.out, "out", "o", "bundle.json", "path to the output bundle.json (- for stdout)")
	return cmd
}

func runBundle(dockerCli command.Cli, appName string, opts bundleOptions) error {
	bundle, err := makeBundle(dockerCli, appName)
	if err != nil {
		return err
	}
	if bundle == nil || len(bundle.InvocationImages) == 0 {
		return fmt.Errorf("failed to create bundle %q", appName)
	}
	fmt.Fprintf(dockerCli.Out(), "Invocation image %q successfully built\n", bundle.InvocationImages[0].Image)
	bundleBytes, err := json.MarshalIndent(bundle, "", "\t")
	if err != nil {
		return err
	}
	if opts.out == "-" {
		_, err = dockerCli.Out().Write(bundleBytes)
		return err
	}
	return ioutil.WriteFile(opts.out, bundleBytes, 0644)
}

func makeBundle(dockerCli command.Cli, appName string) (*bundle.Bundle, error) {
	app, err := packager.Extract(appName)
	if err != nil {
		return nil, err
	}
	defer app.Cleanup()
	return makeBundleFromApp(dockerCli, app)
}

func makeBundleFromApp(dockerCli command.Cli, app *types.App) (*bundle.Bundle, error) {
	meta := app.Metadata()
	invocationImageName, err := makeImageName(meta)
	if err != nil {
		return nil, err
	}
	if _, err := makeImageName(meta); err != nil {
		return nil, err
	}

	buildContext := bytes.NewBuffer(nil)
	if err := packager.PackInvocationImageContext(app, buildContext); err != nil {
		return nil, err
	}

	buildResp, err := dockerCli.Client().ImageBuild(context.TODO(), buildContext, dockertypes.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{invocationImageName},
	})
	if err != nil {
		return nil, err
	}
	defer buildResp.Body.Close()

	if err := jsonmessage.DisplayJSONMessagesStream(buildResp.Body, ioutil.Discard, 0, false, func(jsonmessage.JSONMessage) {}); err != nil {
		return nil, err
	}
	return packager.ToCNAB(app, invocationImageName)
}

func makeImageName(meta metadata.AppMetadata) (string, error) {
	name := fmt.Sprintf("%s:%s-invoc", meta.Name, meta.Version)
	if _, err := reference.ParseNormalizedNamed(name); err != nil {
		return "", errors.Wrapf(err, "image name %q is invalid, please check name and version fields", name)
	}
	return name, nil
}
