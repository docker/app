package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/deis/duffle/pkg/bundle"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/types"
	"github.com/docker/app/types/metadata"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type bundleOptions struct {
	invocationImageName string
	namespace           string
	out                 string
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

	cmd.Flags().StringVarP(&opts.invocationImageName, "invocation-image", "i", "", "specify the name of invocation image to build")
	cmd.Flags().StringVar(&opts.namespace, "namespace", "", "namespace to use (default: namespace in metadata)")
	cmd.Flags().StringVarP(&opts.out, "out", "o", "bundle.json", "path to the output bundle.json (- for stdout)")
	return cmd
}

func runBundle(dockerCli command.Cli, appName string, opts bundleOptions) error {
	bundle, err := makeBundle(dockerCli, appName, opts.namespace, opts.invocationImageName)
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

func makeBundle(dockerCli command.Cli, appName, namespace, invocationImageName string) (*bundle.Bundle, error) {
	app, err := packager.Extract(appName)
	if err != nil {
		return nil, err
	}
	defer app.Cleanup()
	return makeBundleFromApp(dockerCli, app, namespace, invocationImageName)
}

func resolveAuthConfigsForBuild(dockerCli command.Cli) (map[string]dockertypes.AuthConfig, error) {
	ref, err := reference.ParseNormalizedNamed("docker/cnab-app-base")
	if err != nil {
		return nil, err
	}
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return nil, err
	}
	return map[string]dockertypes.AuthConfig{
		registry.GetAuthConfigKey(repoInfo.Index): command.ResolveAuthConfig(context.TODO(), dockerCli, repoInfo.Index),
	}, nil
}

func makeBundleFromApp(dockerCli command.Cli, app *types.App, namespace, invocationImageName string) (*bundle.Bundle, error) {
	meta := app.Metadata()
	invocationImageName, err := makeInvocationImageName(meta, namespace, invocationImageName)
	if err != nil {
		return nil, err
	}
	if err := checkAppImage(app.Metadata(), namespace); err != nil {
		return nil, err
	}

	buildContext := bytes.NewBuffer(nil)
	if err := packager.PackInvocationImageContext(app, buildContext); err != nil {
		return nil, err
	}

	authConfigs, err := resolveAuthConfigsForBuild(dockerCli)
	if err != nil {
		return nil, err
	}

	buildResp, err := dockerCli.Client().ImageBuild(context.TODO(), buildContext, dockertypes.ImageBuildOptions{
		Dockerfile:  "Dockerfile",
		Tags:        []string{invocationImageName},
		AuthConfigs: authConfigs,
	})
	if err != nil {
		return nil, err
	}
	defer buildResp.Body.Close()

	if err := jsonmessage.DisplayJSONMessagesStream(buildResp.Body, ioutil.Discard, 0, false, func(jsonmessage.JSONMessage) {}); err != nil {
		return nil, err
	}
	return packager.ToCNAB(app, invocationImageName), nil
}

func makeInvocationImageName(meta metadata.AppMetadata, namespace, name string) (string, error) {
	if name == "" {
		name = fmt.Sprintf("%s:%s-invoc", meta.Name, meta.Version)
	}
	ns := namespace
	if ns == "" {
		ns = meta.Namespace
	}
	if ns != "" {
		name = fmt.Sprintf("%s/%s", ns, name)
	}
	if _, err := reference.ParseNormalizedNamed(name); err != nil {
		return "", errors.Wrapf(err, "invocation image name %q is invalid", name)
	}
	return name, nil
}

func checkAppImage(meta metadata.AppMetadata, namespace string) error {
	name := fmt.Sprintf("%s:%s", meta.Name, meta.Version)
	ns := namespace
	if ns == "" {
		ns = meta.Namespace
	}
	if ns != "" {
		name = fmt.Sprintf("%s/%s", ns, name)
	}
	_, err := reference.ParseNormalizedNamed(name)
	return errors.Wrapf(err, "generated app image name %q is invalid, please check namespace, name and version fields", name)
}
