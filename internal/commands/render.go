package commands

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/deislabs/duffle/pkg/driver"
	"github.com/docker/app/internal"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/spf13/cobra"
)

type renderOptions struct {
	parametersOptions
	registryOptions
	pullOptions

	formatDriver string
	renderOutput string
}

func renderCmd(dockerCli command.Cli) *cobra.Command {
	var opts renderOptions
	cmd := &cobra.Command{
		Use:     "render [APP_NAME] [--set KEY=VALUE ...] [--parameters-file PARAMETERS-FILE ...] [OPTIONS]",
		Short:   "Render the Compose file for an Application Package",
		Example: `$ docker app render myapp.dockerapp --set key=value`,
		Args:    cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRender(dockerCli, firstOrEmpty(args), opts)
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
	opts.registryOptions.addFlags(cmd.Flags())
	opts.pullOptions.addFlags(cmd.Flags())
	cmd.Flags().StringVarP(&opts.renderOutput, "output", "o", "-", "Output file")
	cmd.Flags().StringVar(&opts.formatDriver, "formatter", "yaml", "Configure the output format (yaml|json)")

	return cmd
}

func runRender(dockerCli command.Cli, appname string, opts renderOptions) error {
	defer muteDockerCli(dockerCli)()

	var w io.Writer = os.Stdout
	if opts.renderOutput != "-" {
		f, err := os.Create(opts.renderOutput)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	action, installation, errBuf, err := prepareCustomAction(internal.ActionRenderName, dockerCli, appname, w, opts.registryOptions, opts.pullOptions, opts.parametersOptions)
	if err != nil {
		return err
	}
	installation.Parameters[internal.ParameterRenderFormatName] = opts.formatDriver

	if dir, ok := os.LookupEnv("INVOC_STRACE_DIR"); ok {
		d := action.Driver.(*driver.DockerDriver)
		d.AddConfigurationOptions(
			func(config *container.Config, hostConfig *container.HostConfig) error {
				fmt.Fprintf(os.Stderr, "Stracing invoc image to %q\n", dir)
				config.User = "0:0"
				//fmt.Fprintf(os.Stderr, "Original entrypoint is %+v\n", config.Entrypoint)
				config.Entrypoint = append([]string{"strace", "-s", "4096", "-fff", "-o", "/strace/cnab-run.render"}, config.Entrypoint...)
				//fmt.Fprintf(os.Stderr, "New entrypoint is %+v\n", config.Entrypoint)

				m := mount.Mount{
					Type:   mount.TypeBind,
					Source: dir,
					Target: "/strace",
				}
				//fmt.Fprintf(os.Stderr, "Mount: %+v\n", m)
				hostConfig.Mounts = append(hostConfig.Mounts, m)

				//fmt.Fprintf(os.Stderr, "Original CapAdd: %+v\n", hostConfig.CapAdd)
				hostConfig.CapAdd = append(hostConfig.CapAdd, "SYS_PTRACE")
				//fmt.Fprintf(os.Stderr, "New CapAdd: %+v\n", hostConfig.CapAdd)
				return nil
			},
		)
	}

	fmt.Fprintf(os.Stderr, "%s\n", time.Now())
	fmt.Fprintf(os.Stderr, "Rendering %q using format %q\n", appname, opts.formatDriver)
	fmt.Fprintf(os.Stderr, "Action: %+v\n", action)
	fmt.Fprintf(os.Stderr, "Installation: %+v\n", installation)
	if err := action.Run(&installation.Claim, nil, nil); err != nil {
		return fmt.Errorf("render failed: %s", errBuf)
	}
	fmt.Fprintf(os.Stderr, "%s: START RENDER STDERR:\n", time.Now())
	fmt.Fprintf(os.Stderr, "%s\n", errBuf)
	fmt.Fprintf(os.Stderr, "%s: END RENDER STDERR\n", time.Now())
	return nil
}
