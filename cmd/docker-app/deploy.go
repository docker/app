package main

import (
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/image"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/render"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type deployOptions struct {
	deployComposeFiles     []string
	deploySettingsFiles    []string
	deployEnv              []string
	deployOrchestrator     string
	deployKubeConfig       string
	deployNamespace        string
	deployStackName        string
	deploySendRegistryAuth bool
	deployOverrideRegistry string
	deployPushImages       bool
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
	cmd.Flags().StringVarP(&opts.deployOverrideRegistry, "registry", "r", "", "Override registry for all images")
	cmd.Flags().BoolVarP(&opts.deploySendRegistryAuth, "with-registry-auth", "", false, "Sends registry auth")
	cmd.Flags().BoolVarP(&opts.deployPushImages, "push", "", false, "Push images to given registry first")
	if internal.Experimental == "on" {
		cmd.Flags().StringArrayVarP(&opts.deployComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	}
	return cmd
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
	if opts.deployPushImages {
		if err := image.Push(app.Path, opts.deployOverrideRegistry, nil, rendered); err != nil {
			return err
		}
	}
	if opts.deployOverrideRegistry != "" {
		if err = image.ChangeAllImages(rendered, opts.deployOverrideRegistry); err != nil {
			return err
		}
	}
	stackName := opts.deployStackName
	if stackName == "" {
		stackName = internal.AppNameFromDir(app.Name)
	}
	return stack.RunDeploy(dockerCli, flags, rendered, deployOrchestrator, options.Deploy{
		Namespace:        stackName,
		ResolveImage:     swarm.ResolveImageAlways,
		SendRegistryAuth: opts.deploySendRegistryAuth,
	})
}
