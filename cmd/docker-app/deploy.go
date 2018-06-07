package main

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/renderer"
	"github.com/docker/app/internal/utils"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/spf13/cobra"
)

type deployOptions struct {
	deployComposeFiles  []string
	deploySettingsFiles []string
	deployEnv           []string
	deployOrchestrator  string
	deployKubeConfig    string
	deployNamespace     string
	deployStackName     string
}

// deployCmd represents the deploy command
func deployCmd() *cobra.Command {
	var opts deployOptions

	cmd := &cobra.Command{
		Use:   "deploy [<app-name>]",
		Short: "Deploy or update an application",
		Long:  `Deploy the application on either Swarm or Kubernetes.`,
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(firstOrEmpty(args), opts)
		},
	}

	cmd.Flags().StringArrayVarP(&opts.deploySettingsFiles, "settings-files", "f", []string{}, "Override settings files")
	cmd.Flags().StringArrayVarP(&opts.deployEnv, "set", "s", []string{}, "Override settings values")
	cmd.Flags().StringVarP(&opts.deployOrchestrator, "orchestrator", "o", "swarm", "Orchestrator to deploy on (swarm, kubernetes)")
	cmd.Flags().StringVarP(&opts.deployKubeConfig, "kubeconfig", "k", "", "kubeconfig file to use")
	cmd.Flags().StringVarP(&opts.deployNamespace, "namespace", "n", "default", "namespace to deploy into")
	cmd.Flags().StringVarP(&opts.deployStackName, "name", "d", "", "stack name (default: app name)")
	if internal.Experimental == "on" {
		cmd.Flags().StringArrayVarP(&opts.deployComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	}
	return cmd
}

func runDeploy(appname string, opts deployOptions) error {
	deployOrchestrator := opts.deployOrchestrator
	if do, ok := os.LookupEnv("DOCKER_ORCHESTRATOR"); ok {
		deployOrchestrator = do
	}
	if deployOrchestrator != "swarm" && deployOrchestrator != "kubernetes" {
		return fmt.Errorf("orchestrator must be either 'swarm' or 'kubernetes'")
	}
	d, err := parseSettings(opts.deployEnv)
	if err != nil {
		return err
	}
	appname, cleanup, err := packager.Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
	rendered, err := renderer.Render(appname, opts.deployComposeFiles, opts.deploySettingsFiles, d)
	if err != nil {
		return err
	}
	cli := command.NewDockerCli(os.Stdin, os.Stdout, os.Stderr, true)
	cli.Initialize(&cliflags.ClientOptions{
		Common: &cliflags.CommonOptions{
			Orchestrator: deployOrchestrator,
		},
	})
	stackName := opts.deployStackName
	if stackName == "" {
		stackName = utils.AppNameFromDir(appname)
	}
	if deployOrchestrator == "swarm" {
		ctx := context.Background()
		return swarm.DeployCompose(ctx, cli, rendered, options.Deploy{
			Namespace: stackName,
		})
	}
	// kube mode
	kubeCli, err := kubernetes.WrapCli(cli, kubernetes.Options{
		Namespace: opts.deployNamespace,
		Config:    opts.deployKubeConfig,
	})
	if err != nil {
		return err
	}
	return kubernetes.DeployStack(kubeCli, options.Deploy{Namespace: stackName}, rendered)
}
