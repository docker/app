package renderer

import (
	"context"
	"os"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack/kubernetes"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/cli/command/stack/swarm"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/lunchbox/packager"
	"github.com/docker/lunchbox/utils"
)

// Deploy deploys this app, merging in settings files, other compose files, end env
func Deploy(appname string, composeFiles []string, settingsFile []string, env map[string]string,
	orchestrator string, kubeconfig string, namespace string) error {
	appname, cleanup, err := packager.Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
	rendered, err := Render(appname, composeFiles, settingsFile, env)
	if err != nil {
		return err
	}
	cli := command.NewDockerCli(os.Stdin, os.Stdout, os.Stderr, true)
	cli.Initialize(&cliflags.ClientOptions{
		Common: &cliflags.CommonOptions{
			Orchestrator: orchestrator,
		},
	})
	if orchestrator == "swarm" {
		ctx := context.Background()
		return swarm.DeployCompose(ctx, cli, rendered, options.Deploy{
			Namespace: utils.AppNameFromDir(appname),
		})
	}
	// kube mode
	kubeCli, err := kubernetes.WrapCli(cli, kubernetes.Options{
		Namespace: namespace,
		Config:    kubeconfig,
	})
	if err != nil {
		return err
	}
	return kubernetes.DeployStack(kubeCli, options.Deploy{Namespace: utils.AppNameFromDir(appname)}, rendered)
}
