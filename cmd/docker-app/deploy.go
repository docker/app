package main

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	cliopts "github.com/docker/cli/opts"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type deployOptions struct {
	deployComposeFiles       []string
	deploySettingsFiles      []string
	deployEnv                []string
	deployOrchestrator       string
	deployKubeConfig         string
	deployNamespace          string
	deployNoRenderAttachment []string
	deployStackName          string
	deploySendRegistryAuth   bool
}

// deployCmd represents the deploy command
func deployCmd(dockerCli command.Cli) *cobra.Command {
	var opts deployOptions

	cmd := &cobra.Command{
		Use:   "deploy [<app-name>]",
		Short: "Deploy or update an application",
		Long:  `Deploy the application on either Swarm or Kubernetes.`,
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(dockerCli, cmd.Flags(), firstOrEmpty(args), opts)
		},
	}

	cmd.Flags().StringArrayVarP(&opts.deploySettingsFiles, "settings-files", "f", []string{}, "Override settings files")
	cmd.Flags().StringArrayVarP(&opts.deployEnv, "set", "s", []string{}, "Override settings values")
	cmd.Flags().StringVarP(&opts.deployOrchestrator, "orchestrator", "o", "swarm", "Orchestrator to deploy on (swarm, kubernetes)")
	cmd.Flags().StringVarP(&opts.deployKubeConfig, "kubeconfig", "k", "", "Kubernetes config file to use")
	cmd.Flags().StringVarP(&opts.deployNamespace, "namespace", "n", "default", "Kubernetes namespace to deploy into")
	cmd.Flags().StringVarP(&opts.deployStackName, "name", "d", "", "Stack name (default: app name)")
	cmd.Flags().BoolVarP(&opts.deploySendRegistryAuth, "with-registry-auth", "", false, "Sends registry auth")
	if internal.Experimental == "on" {
		cmd.Flags().StringArrayVarP(&opts.deployComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
		cmd.Flags().StringArrayVarP(&opts.deployNoRenderAttachment, "no-render", "", []string{}, "Specify a glob pattern of attachments for which to skip rendering")
	}
	return cmd
}

func skipAttachment(path string, skipPatterns []string) (bool, error) {
	for _, s := range skipPatterns {
		match, err := filepath.Match(s, path)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}

func linkOrCopy(source, target string) error {
	if err := os.Symlink(source, target); err != nil {
		if err := os.Link(source, target); err != nil {
			// fallback to copy
			src, err := os.Open(source)
			if err != nil {
				return err
			}
			dst, err := os.Create(target)
			if err != nil {
				src.Close()
				return err
			}
			if _, err = io.Copy(dst, src); err != nil {
				src.Close()
				dst.Close()
				return errors.Wrapf(err, "failed to copy attachment %s", source)
			}
			src.Close()
			dst.Close()
		}
	}
	return nil
}

func renderAttachments(app *types.App, tempDir string, d map[string]string, skipPatterns []string) error {
	for _, a := range app.Attachments() {
		source := filepath.Join(app.Path, a.Path())
		target := filepath.Join(tempDir, a.Path())
		skip, err := skipAttachment(a.Path(), skipPatterns)
		if err != nil {
			return err
		}
		if skip {
			if err := linkOrCopy(source, target); err != nil {
				return err
			}
			continue
		}
		content, err := ioutil.ReadFile(source)
		if err != nil {
			return err
		}
		rendered, err := render.RenderConfig(app, d, string(content))
		if err != nil {
			return errors.Wrapf(err, "failed to render %s", a.Path())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return errors.Wrap(err, "failed to create temporary directory")
		}
		if err := ioutil.WriteFile(target, []byte(rendered), 0600); err != nil {
			return errors.Wrap(err, "failed to write temporary configuration file")
		}
	}
	return nil
}

func runDeploy(dockerCli command.Cli, flags *pflag.FlagSet, appname string, opts deployOptions) error {
	app, err := packager.Extract(appname,
		types.WithSettingsFiles(opts.deploySettingsFiles...),
		types.WithComposeFiles(opts.deployComposeFiles...),
	)
	if err != nil {
		return err
	}
	defer app.Cleanup()
	deployOrchestrator, err := command.GetStackOrchestrator(opts.deployOrchestrator, dockerCli.ConfigFile().StackOrchestrator, dockerCli.Err())
	if err != nil {
		return err
	}
	d := cliopts.ConvertKVStringsToMap(opts.deployEnv)
	rendered, err := render.Render(app, d)
	if err != nil {
		return err
	}
	stackName := opts.deployStackName
	if stackName == "" {
		stackName = internal.AppNameFromDir(app.Name)
	}
	if len(app.Attachments()) > 0 && internal.Experimental == "on" {
		tempDir, err := ioutil.TempDir("", "app-deploy")
		if err != nil {
			return errors.Wrap(err, "failed to create temporary directory")
		}
		defer os.RemoveAll(tempDir)
		if err := renderAttachments(app, tempDir, d, opts.deployNoRenderAttachment); err != nil {
			return err
		}
		if err := os.Chdir(tempDir); err != nil {
			return err
		}
	} else if app.Source.ShouldRunInsideDirectory() {
		if err := os.Chdir(app.Path); err != nil {
			return err
		}
	}
	return stack.RunDeploy(dockerCli, flags, rendered, deployOrchestrator, options.Deploy{
		Namespace:        stackName,
		ResolveImage:     swarm.ResolveImageAlways,
		SendRegistryAuth: opts.deploySendRegistryAuth,
	})
}
