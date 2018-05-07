package cmd

import (
	"fmt"

	"github.com/docker/cli/cli"
	"github.com/docker/lunchbox/internal"
	"github.com/docker/lunchbox/renderer"
	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy [<app-name>]",
	Short: "Deploy the specified app on the connected cluster",
	Long: `Deploy the application on either swarm or kubernetes.
The app's docker-compose.yml is first rendered as per the render sub-command, and
then deployed similarly to 'docker stack deploy'.`,
	Args: cli.RequiresMaxArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if deployOrchestrator != "swarm" && deployOrchestrator != "kubernetes" {
			return fmt.Errorf("orchestrator must be either 'swarm' or 'kubernetes'")
		}
		d, err := parseSettings(helmEnv)
		if err != nil {
			return err
		}
		return renderer.Deploy(firstOrEmpty(args), deployComposeFiles, deploySettingsFiles, d, deployOrchestrator, deployKubeConfig, deployNamespace)
	},
}

var deployComposeFiles []string
var deploySettingsFiles []string
var deployEnv []string
var deployOrchestrator string
var deployKubeConfig string
var deployNamespace string

func init() {
	if internal.Experimental == "on" {
		rootCmd.AddCommand(deployCmd)
		deployCmd.Flags().StringArrayVarP(&deployComposeFiles, "compose-files", "c", []string{}, "Override Compose files")
		deployCmd.Flags().StringArrayVarP(&deploySettingsFiles, "settings-files", "f", []string{}, "Override settings files")
		deployCmd.Flags().StringArrayVarP(&deployEnv, "set", "s", []string{}, "Override environment values")
		deployCmd.Flags().StringVarP(&deployOrchestrator, "orchestrator", "o", "swarm", "Orchestrator to deploy on (swarm, kubernetes)")
		deployCmd.Flags().StringVarP(&deployKubeConfig, "kubeconfig", "k", "", "kubeconfig file to use")
		deployCmd.Flags().StringVarP(&deployNamespace, "namespace", "n", "default", "namespace to deploy into")

	}
}
