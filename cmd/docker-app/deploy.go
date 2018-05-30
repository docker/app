package main

import (
	"fmt"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/renderer"
	"github.com/spf13/cobra"
)

var (
	deployComposeFiles  []string
	deploySettingsFiles []string
	deployEnv           []string
	deployOrchestrator  string
	deployKubeConfig    string
	deployNamespace     string
	deployStackName     string
)

// deployCmd represents the deploy command
func deployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [<app-name>]",
		Short: "Deploy or update an application",
		Long:  `Deploy the application on either Swarm or Kubernetes.`,
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if do, ok := os.LookupEnv("DOCKER_ORCHESTRATOR"); ok {
				deployOrchestrator = do
			}
			if deployOrchestrator != "swarm" && deployOrchestrator != "kubernetes" {
				return fmt.Errorf("orchestrator must be either 'swarm' or 'kubernetes'")
			}
			d, err := parseSettings(deployEnv)
			if err != nil {
				return err
			}
			return renderer.Deploy(firstOrEmpty(args), deployComposeFiles, deploySettingsFiles, d, deployStackName, deployOrchestrator, deployKubeConfig, deployNamespace)
		},
	}

	cmd.Flags().StringArrayVarP(&deploySettingsFiles, "settings-files", "f", []string{}, "Override settings files")
	cmd.Flags().StringArrayVarP(&deployEnv, "set", "s", []string{}, "Override settings values")
	cmd.Flags().StringVarP(&deployOrchestrator, "orchestrator", "o", "swarm", "Orchestrator to deploy on (swarm, kubernetes)")
	cmd.Flags().StringVarP(&deployKubeConfig, "kubeconfig", "k", "", "kubeconfig file to use")
	cmd.Flags().StringVarP(&deployNamespace, "namespace", "n", "default", "namespace to deploy into")
	cmd.Flags().StringVarP(&deployStackName, "name", "d", "", "stack name (default: app name)")
	if internal.Experimental == "on" {
		cmd.Flags().StringArrayVarP(&deployComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	}
	return cmd
}
