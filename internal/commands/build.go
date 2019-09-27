package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/app/internal/packager"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/buildx/driver"

	"github.com/docker/buildx/util/progress"

	cnab "github.com/deislabs/cnab-go/driver"
	"github.com/docker/buildx/bake"
	"github.com/docker/buildx/build"
	_ "github.com/docker/buildx/driver/docker" // required to get default driver registered, see driver/docker/factory.go:14
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	dockerclient "github.com/docker/docker/client"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/spf13/cobra"
)

type buildOptions struct {
	noCache  bool
	progress string
	pull     bool
	tag      string
}

func buildCmd(dockerCli command.Cli) *cobra.Command {
	var opts buildOptions
	cmd := &cobra.Command{
		Use:     "build [APPLICATION]",
		Short:   "Build service images for the application",
		Example: `$ docker app build myapp.dockerapp`,
		Args:    cli.RequiresRangeArgs(1, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(dockerCli, args[0], opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.noCache, "no-cache", false, "Do not use cache when building the image")
	flags.StringVar(&opts.progress, "progress", "auto", "Set type of progress output (auto, plain, tty). Use plain to show container output")
	flags.BoolVar(&opts.pull, "pull", false, "Always attempt to pull a newer version of the image")
	cmd.Flags().StringVarP(&opts.tag, "tag", "t", "", "Name and optionally a tag in the 'name:tag' format")

	return cmd
}

func runBuild(dockerCli command.Cli, application string, opt buildOptions) error {
	app, err := packager.Extract(application)
	if err != nil {
		return err
	}
	defer app.Cleanup()
	appname := app.Name

	bundle, err := makeBundleFromApp(dockerCli, app, nil)
	if err != nil {
		return err
	}

	ctx := appcontext.Context()
	compose, err := bake.ParseCompose(app.Composes()[0]) // Fixme can have > 1 composes ?
	if err != nil {
		return err
	}

	targets := map[string]bake.Target{}
	for _, n := range compose.ResolveGroup("default") {
		t, err := compose.ResolveTarget(n)
		if err != nil {
			return nil
		}
		if t != nil {
			targets[n] = *t
		}
	}

	for service, t := range targets {
		if strings.HasPrefix(*t.Context, ".") {
			// Relative path in compose file under x.dockerapp refers to parent folder
			// FIXME docker app init should maybe udate them ?
			path, err := filepath.Abs(appname + "/../" + (*t.Context)[1:])
			if err != nil {
				return err
			}
			t.Context = &path
			t.Tags = []string{fmt.Sprintf("%s:%s-%s", bundle.Name, bundle.Version, service)}
			targets[service] = t
		}
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel)	{
		dt, err := json.MarshalIndent(map[string]map[string]bake.Target{"target": targets}, "", "   ")
		if err != nil {
			return err
		}
		logrus.Debug(string(dt))
	}

	buildopts, err := bake.TargetsToBuildOpt(targets, opt.noCache, opt.pull)
	if err != nil {
		return err
	}

	buildContext := bytes.NewBuffer(nil)
	if err := packager.PackInvocationImageContext(dockerCli, app, buildContext); err != nil {
		return err
	}

	buildopts["invocation-image"] = build.Options{
		Inputs:      build.Inputs{
			InStream: buildContext,
			ContextPath: "-",
		},
		Tags:        []string{ fmt.Sprintf("%s:%s-%s", bundle.Name, bundle.Version, "-invoc") },
		Session:     []session.Attachable{authprovider.NewDockerAuthProvider(os.Stderr)},
	}

	d, err := driver.GetDriver(ctx, "buildx_buildkit_default", nil, dockerCli.Client(), nil, "", nil)
	if err != nil {
		return err
	}
	driverInfo := []build.DriverInfo{
		{
			Name:   "default",
			Driver: d,
		},
	}

	ctx2, cancel := context.WithCancel(context.TODO())
	defer cancel()

	pw := progress.NewPrinter(ctx2, os.Stderr, opt.progress)
	resp, err := build.Build(ctx2, driverInfo, buildopts, dockerAPI(dockerCli), dockerCli.ConfigFile(), pw)
	if err != nil {
		return err
	}

	fmt.Println("Successfully built service images")

	for service, r := range resp {
		digest := r.ExporterResponse["containerimage.digest"]
		if service == "invocation-image" {
			bundle.InvocationImages[0].Digest = digest
		} else {
			image := bundle.Images[service]
			image.Image = fmt.Sprintf("%s:%s-%s", bundle.Name, bundle.Version, service)
			image.ImageType = cnab.ImageTypeDocker
			image.Digest = digest
			bundle.Images[service] = image
		}
		fmt.Printf("    - %s : %s\n", service, digest)
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		dt, err := json.MarshalIndent(resp, "", "   ")
		if err != nil {
			return err
		}
		logrus.Debug(string(dt))
	}

	if opt.tag == "" {
		opt.tag = bundle.Name + ":" + bundle.Version
	}

	ref, err := getNamedTagged(opt.tag)
	if err != nil {
		return err
	}

	if err := persistInBundleStore(ref, bundle); err != nil {
		return err
	}

	return nil
}

/// FIXME copy from vendor/github.com/docker/buildx/commands/util.go:318 could probably be made public
func dockerAPI(dockerCli command.Cli) *api {
	return &api{dockerCli: dockerCli}
}

type api struct {
	dockerCli command.Cli
}

func (a *api) DockerAPI(name string) (dockerclient.APIClient, error) {
	if name == "" {
		name = a.dockerCli.CurrentContext()
	}
	return nil, fmt.Errorf("Only support default context in this prototype")
}
