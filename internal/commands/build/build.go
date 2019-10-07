package build

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
	cnab "github.com/deislabs/cnab-go/driver"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/types"
	"github.com/docker/buildx/build"
	"github.com/docker/buildx/driver"
	_ "github.com/docker/buildx/driver/docker" // required to get default driver registered, see driver/docker/factory.go:14
	"github.com/docker/buildx/util/progress"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type buildOptions struct {
	noCache  bool
	progress string
	pull     bool
	tag      string
}

func Cmd(dockerCli command.Cli) *cobra.Command {
	var opts buildOptions
	cmd := &cobra.Command{
		Use:     "build [APP_NAME] [APP_IMAGE]",
		Short:   "Build service images for the application",
		Example: `$ docker app build myapp.dockerapp my/app:1.0.0`,
		Args:    cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.tag = args[1]
			tag, err := runBuild(dockerCli, args[0], opts)
			if err == nil {
				fmt.Printf("Successfully build %s\n", tag.String())
			}
			return err
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.noCache, "no-cache", false, "Do not use cache when building the image")
	flags.StringVar(&opts.progress, "progress", "auto", "Set type of progress output (auto, plain, tty). Use plain to show container output")
	flags.BoolVar(&opts.pull, "pull", false, "Always attempt to pull a newer version of the image")

	return cmd
}

func runBuild(dockerCli command.Cli, application string, opt buildOptions) (reference.Named, error) {
	err := checkMinimalEngineVersion(dockerCli)
	if err != nil {
		return nil, err
	}

	if opt.tag == "" {
		// FIXME temporary, until we get support for Digest in bundleStore and other commands
		return nil, fmt.Errorf("A tag is required to run docker app build")
	}

	var ref reference.Named
	ref, err = packager.GetNamedTagged(opt.tag)
	if err != nil {
		return nil, err
	}

	app, err := packager.Extract(application)
	if err != nil {
		return nil, err
	}
	defer app.Cleanup()

	buildopts, err := parseCompose(app, opt)
	if err != nil {
		return nil, err
	}

	buildopts["com.docker.app.invocation-image"], err = createInvocationImageBuildOptions(dockerCli, app)
	if err != nil {
		return nil, err
	}

	debugBuildOpts(buildopts)

	ctx, cancel := context.WithCancel(appcontext.Context())
	defer cancel()
	const drivername = "buildx_buildkit_default"
	d, err := driver.GetDriver(ctx, drivername, nil, dockerCli.Client(), nil, "", nil)
	if err != nil {
		return nil, err
	}
	driverInfo := []build.DriverInfo{
		{
			Name:   "default",
			Driver: d,
		},
	}

	pw := progress.NewPrinter(ctx, os.Stderr, opt.progress)

	// We rely on buildx "docker" builder integrated in docker engine, so don't need a DockerAPI here
	resp, err := build.Build(ctx, driverInfo, buildopts, nil, dockerCli.ConfigFile(), pw)
	if err != nil {
		return nil, err
	}
	fmt.Fprintln(dockerCli.Out(), "Successfully built service images") //nolint:errcheck

	bundle, err := packager.MakeBundleFromApp(dockerCli, app, nil)
	if err != nil {
		return nil, err
	}
	err = updateBundle(dockerCli, bundle, resp)
	if err != nil {
		return nil, err
	}

	if ref == nil {
		if ref, err = computeDigest(bundle); err != nil {
			return nil, err
		}
	}

	if err = packager.PersistInBundleStore(ref, bundle); err != nil {
		return nil, err
	}
	return ref, nil
}

func checkMinimalEngineVersion(dockerCli command.Cli) error {
	info, err := dockerCli.Client().Info(appcontext.Context())
	if err != nil {
		return err
	}
	majorVersion, err := strconv.Atoi(strings.SplitN(info.ServerVersion, ".", 2)[0])
	if err != nil {
		return err
	}
	if majorVersion < 19 {
		return errors.New("'build' require docker engine 19.03 or later")
	}
	return nil
}

func updateBundle(dockerCli command.Cli, bundle *bundle.Bundle, resp map[string]*client.SolveResponse) error {
	debugSolveResponses(resp)
	for service, r := range resp {
		digest := r.ExporterResponse["containerimage.digest"]
		inspect, _, err := dockerCli.Client().ImageInspectWithRaw(context.TODO(), digest)
		if err != nil {
			return err
		}
		size := uint64(inspect.Size)
		if service == "com.docker.app.invocation-image" {
			bundle.InvocationImages[0].Digest = digest
			bundle.InvocationImages[0].Size = size
		} else {
			image := bundle.Images[service]
			image.ImageType = cnab.ImageTypeDocker
			image.Digest = digest
			image.Size = size
			bundle.Images[service] = image
		}
	}
	debugBundle(bundle)
	return nil
}

func createInvocationImageBuildOptions(dockerCli command.Cli, app *types.App) (build.Options, error) {
	buildContext := bytes.NewBuffer(nil)
	if err := packager.PackInvocationImageContext(dockerCli, app, buildContext); err != nil {
		return build.Options{}, err
	}
	return build.Options{
		Inputs: build.Inputs{
			InStream:    buildContext,
			ContextPath: "-",
		},
		Session: []session.Attachable{authprovider.NewDockerAuthProvider(os.Stderr)},
	}, nil
}

func debugBuildOpts(opts map[string]build.Options) {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		dt, err := json.MarshalIndent(opts, "  > ", "   ")
		if err != nil {
			logrus.Debugf("Failed to marshal Bundle: %s", err.Error())
		} else {
			logrus.Debug(string(dt))
		}
	}
}

func debugBundle(bundle *bundle.Bundle) {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		dt, err := json.MarshalIndent(bundle, "  > ", "   ")
		if err != nil {
			logrus.Debugf("Failed to marshal Bundle: %s", err.Error())
		} else {
			logrus.Debug(string(dt))
		}
	}
}

func debugSolveResponses(resp map[string]*client.SolveResponse) {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		dt, err := json.MarshalIndent(resp, "  > ", "   ")
		if err != nil {
			logrus.Debugf("Failed to marshal Buildx response: %s", err.Error())
		} else {
			logrus.Debug(string(dt))
		}
	}
}
