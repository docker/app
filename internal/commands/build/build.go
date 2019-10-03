package build

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
	out      string
}

func Cmd(dockerCli command.Cli) *cobra.Command {
	var opts buildOptions
	cmd := &cobra.Command{
		Use:     "build [APPLICATION]",
		Short:   "Build service images for the application",
		Example: `$ docker app build myapp.dockerapp`,
		Args:    cli.ExactArgs(1),
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
	ctx := appcontext.Context()
	info, err := dockerCli.Client().Info(ctx)
	if err != nil {
		return nil, err
	}
	majorVersion, err := strconv.Atoi(info.ServerVersion[:strings.IndexRune(info.ServerVersion, '.')])
	if err != nil {
		return nil, err
	}
	if majorVersion < 19 {
		return nil, errors.New("'build' require docker engine 19.03 or later")
	}

	app, err := packager.Extract(application)
	if err != nil {
		return nil, err
	}
	defer app.Cleanup()

	bundle, err := packager.MakeBundleFromApp(dockerCli, app, nil)
	if err != nil {
		return nil, err
	}

	buildopts, err := parseCompose(app, opt)
	if err != nil {
		return nil, err
	}

	buildopts["invocation-image"], err = createInvocationImageBuildOptions(dockerCli, app)
	if err != nil {
		return nil, err
	}

	debugBuildOpts(buildopts)

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

	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	pw := progress.NewPrinter(ctx2, os.Stderr, opt.progress)

	// We rely on buildx "docker" builder integrated in docker engine, so don't nee a DockerAPI here
	resp, err := build.Build(ctx2, driverInfo, buildopts, nil, dockerCli.ConfigFile(), pw)
	if err != nil {
		return nil, err
	}

	fmt.Println("Successfully built service images")
	updateBundle(bundle, resp)

	var ref reference.Named
	ref, err = packager.GetNamedTagged(opt.tag)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		if ref, err = computeDigest(bundle); err != nil {
			return nil, err
		}
	}

	if err = persistBundle(opt, bundle, ref); err != nil {
		return nil, err
	}
	return ref, nil
}

func updateBundle(bundle *bundle.Bundle, resp map[string]*client.SolveResponse) {
	debugSolveResponses(resp)
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
	}
	debugBundle(bundle)
}

func persistBundle(opt buildOptions, bndl *bundle.Bundle, ref reference.Named) error {
	if opt.out != "" {
		b, err := json.MarshalIndent(bndl, "", "  ")
		if err != nil {
			return err
		}
		if opt.out == "-" {
			_, err = os.Stdout.Write(b)
			if err != nil {
				return err
			}
		} else {
			err = ioutil.WriteFile(opt.out, b, 0644)
			if err != nil {
				return err
			}
		}
	} else {
		if err := packager.PersistInBundleStore(ref, bndl); err != nil {
			return err
		}
	}
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
