package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/app/internal/packager"
	"github.com/docker/distribution/reference"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
	"io/ioutil"
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
	out		 string
}

func buildCmd(dockerCli command.Cli) *cobra.Command {
	var opts buildOptions
	cmd := &cobra.Command{
		Use:     "build [APPLICATION]",
		Short:   "Build service images for the application",
		Example: `$ docker app build myapp.dockerapp`,
		Args:    cli.RequiresRangeArgs(1, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
	flags.StringVarP(&opts.out, "output", "o", "", "Dump generated bundle into a file")
	flags.StringVarP(&opts.tag, "tag", "t", "", "Name and optionally a tag in the 'name:tag' format")

	return cmd
}

func runBuild(dockerCli command.Cli, application string, opt buildOptions) (reference.Named, error) {
	app, err := packager.Extract(application)
	if err != nil {
		return nil, err
	}
	defer app.Cleanup()
	appname := app.Name

	bundle, err := makeBundleFromApp(dockerCli, app, nil)
	if err != nil {
		return nil, err
	}

	ctx := appcontext.Context()

	compose, err := bake.ParseCompose(app.Composes()[0]) // Fixme can have > 1 composes ?
	if err != nil {
		return nil, err
	}

	targets := map[string]bake.Target{}
	for _, n := range compose.ResolveGroup("default") {
		t, err := compose.ResolveTarget(n)
		if err != nil {
			return nil, err
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
				return nil, err
			}
			t.Context = &path
			targets[service] = t
		}
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel)	{
		dt, err := json.MarshalIndent(targets, "", "   ")
		if err != nil {
			return nil, err
		}
		logrus.Debug(string(dt))
	}

	buildopts, err := bake.TargetsToBuildOpt(targets, opt.noCache, opt.pull)
	if err != nil {
		return nil, err
	}

	buildContext := bytes.NewBuffer(nil)
	if err := packager.PackInvocationImageContext(dockerCli, app, buildContext); err != nil {
		return nil, err
	}

	buildopts["invocation-image"] = build.Options{
		Inputs:      build.Inputs{
			InStream: buildContext,
			ContextPath: "-",
		},
		Session:     []session.Attachable{authprovider.NewDockerAuthProvider(os.Stderr)},
	}

	d, err := driver.GetDriver(ctx, "buildx_buildkit_default", nil, dockerCli.Client(), nil, "", nil)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	fmt.Println("Successfully built service images")
	if logrus.IsLevelEnabled(logrus.DebugLevel)	{
		dt, err := json.MarshalIndent(resp, "", "   ")
		if err != nil {
			return nil, err
		}
		logrus.Debug(string(dt))
	}

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
			return nil, err
		}
		logrus.Debug(string(dt))
	}

	var ref reference.Named
	ref, err = getNamedTagged(opt.tag)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		b := bytes.Buffer{}
		_, err := bundle.WriteTo(&b)
		if err != nil {
			return nil, err
		}
		digest := digest.SHA256.FromBytes(b.Bytes())
		ref	= sha{digest}
	}


	if opt.out != "" {
		b, err := json.MarshalIndent(bundle, "", "  ")
		if err != nil {
			return ref, err
		}
		if opt.out == "-" {
			_, err = os.Stdout.Write(b)
		} else {
			err = ioutil.WriteFile(opt.out, b, 0644)
		}
		return ref, err
	}

	if err := persistInBundleStore(ref, bundle); err != nil {
		return ref, err
	}

	return ref, nil
}

type sha struct {
	d digest.Digest
}

func (s sha) Digest() digest.Digest {
	return s.d
}

func (s sha) String() string {
	return s.d.String()
}

func (s sha) Name() string {
	return s.d.String()
}

var _ reference.Named = sha{""}
var _ reference.Digested = sha{""}


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
