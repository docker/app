package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/types"
	"github.com/docker/buildx/driver"
	"github.com/docker/buildx/util/progress"
	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/distribution/reference"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	cnab "github.com/deislabs/cnab-go/driver"
	"github.com/docker/buildx/build"
	_ "github.com/docker/buildx/driver/docker" // required to get default driver registered, see driver/docker/factory.go:14
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	composetypes "github.com/docker/cli/cli/compose/types"
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

	_, err = createInvocationImageBuildOptions(dockerCli, app)
	if err != nil {
		return nil, err
	}

	debugBuildOpts(buildopts)

	ctx := appcontext.Context()
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
		fmt.Fprintf(dockerCli.Out(), "    - %s : %s\n", service, digest)
	}
	debugBundle(bundle)

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

	if err := packager.PersistInBundleStore(ref, bundle); err != nil {
		return ref, err
	}

	return ref, nil
}

func computeDigest(bundle *bundle.Bundle) (reference.Named, error) {
	b := bytes.Buffer{}
	_, err := bundle.WriteTo(&b)
	if err != nil {
		return nil, err
	}
	digest := digest.SHA256.FromBytes(b.Bytes())
	ref := sha{digest}
	return ref, nil
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

// parseCompose do parse app compose file and extract buildx targets
func parseCompose(app *types.App, options buildOptions) (map[string]build.Options, error) {

	// Fixme can have > 1 composes ?
	parsed, err := loader.ParseYAML(app.Composes()[0])
	if err != nil {
		return nil, err
	}
	compose, err := loader.Load(composetypes.ConfigDetails{
		ConfigFiles: []composetypes.ConfigFile{
			{
				Config: parsed,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	opts := map[string]build.Options{}
	for _, service := range compose.Services {
		b := service.Build
		if b.Context == "" {
			continue
		}

		if b.Context, err = filepath.Abs(b.Context); err != nil {
			return nil, err
		}
		dockerfilePath := "Dockerfile"
		if b.Dockerfile != "" {
			dockerfilePath = b.Dockerfile
		}
		dockerfilePath = path.Join(b.Context, dockerfilePath)
		opts[service.Name] = build.Options{
			Inputs: build.Inputs{
				ContextPath:    b.Context,
				DockerfilePath: dockerfilePath,
			},
			// BuildArgs: t.Args, //FIXME introduce build args support in docker app build
			NoCache:   options.noCache,
			Pull:      options.pull,
		}
	}
	return opts, nil
}


type sha struct {
	d digest.Digest
}
var _ reference.Named = sha{""}
var _ reference.Digested = sha{""}

// Digest implement Digested.Digest()
func (s sha) Digest() digest.Digest {
	return s.d
}

// Digest implement Named.String()
func (s sha) String() string {
	return s.d.String()
}

// Digest implement Named.Name()
func (s sha) Name() string {
	return s.d.String()
}

