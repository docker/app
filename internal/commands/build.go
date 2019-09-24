package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/buildx/driver"

	"github.com/docker/buildx/util/progress"

	cnab "github.com/deislabs/cnab-go/driver"
	"github.com/docker/app/internal"
	"github.com/docker/buildx/bake"
	"github.com/docker/buildx/build"
	_ "github.com/docker/buildx/driver/docker" // required to get default driver registered, see driver/docker/factory.go:14
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	dockerclient "github.com/docker/docker/client"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/pkg/errors"
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
	appname := internal.DirNameFromAppName(application)
	f := "./" + appname + "/" + internal.ComposeFileName
	if _, err := os.Stat(f); err != nil {
		if os.IsNotExist(errors.Cause(err)) {
			return fmt.Errorf("no compose file at %s, did you selected the right docker app folder ?", f)
		}
	}

	bundle, err := makeBundle(dockerCli, application, nil)
	if err != nil {
		return err
	}

	ctx := appcontext.Context()
	targets, err := bake.ReadTargets(ctx, []string{f}, []string{"default"}, nil)
	if err != nil {
		return err
	}

	for k, t := range targets {
		if strings.HasPrefix(*t.Context, ".") {
			// Relative path in compose file under x.dockerapp refers to parent folder
			// FIXME docker app init should maybe udate them ?
			path, err := filepath.Abs(appname + "/../" + (*t.Context)[1:])
			if err != nil {
				return err
			}
			t.Context = &path
			t.Tags = []string{fmt.Sprintf("%s:%s", bundle.Name, bundle.Version)}
			targets[k] = t
		}
	}

	// -- debug
	dt, err := json.MarshalIndent(map[string]map[string]bake.Target{"target": targets}, "", "   ")
	if err != nil {
		return err
	}
	fmt.Fprintln(dockerCli.Out(), string(dt))
	// -- debug

	buildopts, err := bake.TargetsToBuildOpt(targets, opt.noCache, opt.pull)
	if err != nil {
		return err
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

	// FIXME add invocation image as another build target

	pw := progress.NewPrinter(ctx2, os.Stderr, opt.progress)
	resp, err := build.Build(ctx2, driverInfo, buildopts, dockerAPI(dockerCli), dockerCli.ConfigFile(), pw)
	if err != nil {
		return err
	}

	fmt.Println("Successfully built service images")
	for k, r := range resp {
		digest := r.ExporterResponse["containerimage.digest"]
		image := bundle.Images[k]
		image.ImageType = cnab.ImageTypeDocker
		image.Digest = digest
		bundle.Images[k] = image
		fmt.Printf("    - %s : %s\n", k, image.Digest)
	}

	// -- debug
	dt, err = json.MarshalIndent(resp, "", "   ")
	if err != nil {
		return err
	}
	fmt.Fprintln(dockerCli.Out(), string(dt))
	// -- debug

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
