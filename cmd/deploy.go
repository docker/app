package cmd

import (
	"fmt"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/renderer"
	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
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

var deployComposeFiles []string
var deploySettingsFiles []string
var deployEnv []string
var deployOrchestrator string
var deployKubeConfig string
var deployNamespace string
var deployStackName string

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().StringArrayVarP(&deploySettingsFiles, "settings-files", "f", []string{}, "Override settings files")
	deployCmd.Flags().StringArrayVarP(&deployEnv, "set", "s", []string{}, "Override settings values")
	deployCmd.Flags().StringVarP(&deployOrchestrator, "orchestrator", "o", "swarm", "Orchestrator to deploy on (swarm, kubernetes)")
	deployCmd.Flags().StringVarP(&deployKubeConfig, "kubeconfig", "k", "", "kubeconfig file to use")
	deployCmd.Flags().StringVarP(&deployNamespace, "namespace", "n", "default", "namespace to deploy into")
	deployCmd.Flags().StringVarP(&deployStackName, "name", "d", "", "stack name (default: app name)")
	if internal.Experimental == "on" {
		deployCmd.Flags().StringArrayVarP(&deployComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
	}
}
