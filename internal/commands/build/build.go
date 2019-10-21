package build

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	noCache     bool
	progress    string
	pull        bool
	tag         string
	folder      string
	imageIDFile string
	args        []string
}

func Cmd(dockerCli command.Cli) *cobra.Command {
	var opts buildOptions
	cmd := &cobra.Command{
		Use:     "build [OPTIONS] [CONTEXT_PATH]",
		Short:   "Build service images for the application",
		Example: `$ docker app build --tag my/app:1.0.0 .`,
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBuild(dockerCli, args[0], opts)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.noCache, "no-cache", false, "Do not use cache when building the image")
	flags.StringVar(&opts.progress, "progress", "auto", "Set type of progress output (auto, plain, tty). Use plain to show container output")
	flags.StringVarP(&opts.tag, "tag", "t", "", "Application image and optionally a tag in the 'image:tag' format")
	flags.StringVarP(&opts.folder, "folder", "f", "", "Docker app folder containing application definition")
	flags.BoolVar(&opts.pull, "pull", false, "Always attempt to pull a newer version of the image")
	flags.StringArrayVar(&opts.args, "build-arg", []string{}, "Set build-time variables")
	flags.StringVar(&opts.imageIDFile, "iidfile", "", "Write the app image ID to the file")

	return cmd
}

func runBuild(dockerCli command.Cli, contextPath string, opt buildOptions) error {
	err := checkMinimalEngineVersion(dockerCli)
	if err != nil {
		return err
	}

	if opt.imageIDFile != "" {
		// Avoid leaving a stale file if we eventually fail
		if err := os.Remove(opt.imageIDFile); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	if err = checkBuildArgsUniqueness(opt.args); err != nil {
		return err
	}

	application, err := getAppFolder(opt, contextPath)
	if err != nil {
		return err
	}

	app, err := packager.Extract(application)
	if err != nil {
		return err
	}
	defer app.Cleanup()

	bundle, err := buildImageUsingBuildx(app, contextPath, opt, dockerCli)
	if err != nil {
		return err
	}

	var ref reference.Reference
	ref, err = packager.GetNamedTagged(opt.tag)
	if err != nil {
		return err
	}

	id, err := packager.PersistInBundleStore(ref, bundle)
	if err != nil {
		return err
	}

	if opt.imageIDFile != "" {
		if err = ioutil.WriteFile(opt.imageIDFile, []byte(id.String()), 0644); err != nil {
			fmt.Fprintf(dockerCli.Err(), "Failed to write application image id in %s: %s", opt.imageIDFile, err)
		}
	}

	fmt.Printf("Successfully built %s\n", id)
	if ref != nil {
		fmt.Printf("Successfully tagged %s\n", ref.String())
	}
	return err
}

func buildImageUsingBuildx(app *types.App, contextPath string, opt buildOptions, dockerCli command.Cli) (*bundle.Bundle, error) {
	buildopts, err := parseCompose(app, contextPath, opt)
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
	return bundle, nil
}

func getAppFolder(opt buildOptions, contextPath string) (string, error) {
	application := opt.folder
	if application == "" {
		files, err := ioutil.ReadDir(contextPath)
		if err != nil {
			return "", err
		}
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".dockerapp") {
				if application != "" {
					return "", fmt.Errorf("%s contains multiple *.dockerapp folders, use -f option to select the one to build", contextPath)
				}
				application = filepath.Join(contextPath, f.Name())
				if !f.IsDir() {
					return "", fmt.Errorf("%s isn't a directory", f.Name())
				}
			}
		}
	}
	return application, nil
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

func checkBuildArgsUniqueness(args []string) error {
	set := make(map[string]bool)
	for _, value := range args {
		key := strings.Split(value, "=")[0]
		if _, ok := set[key]; ok {
			return fmt.Errorf("'--build-arg %s' is defined twice", key)
		}
		set[key] = true
	}
	return nil
}
