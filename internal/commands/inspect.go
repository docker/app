package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/deislabs/cnab-go/action"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/cnab"
	"github.com/docker/app/internal/inspect"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack"
	"github.com/docker/cli/cli/command/stack/options"
	"github.com/docker/cli/opts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type inspectOptions struct {
	credentialOptions
	pretty        bool
	orchestrator  string
	kubeNamespace string
}

func inspectCmd(dockerCli command.Cli) *cobra.Command {
	var opts inspectOptions
	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] RUNNING_APP",
		Short: "Shows installation and application metadata, parameters and the containers list of a running application",
		Example: `$ docker app inspect my-running-app
$ docker app inspect my-running-app:1.0.0`,
		Args:   cli.RequiresMaxArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(dockerCli, firstOrEmpty(args), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.pretty, "pretty", false, "Pretty print the output")
	cmd.Flags().StringVar(&opts.orchestrator, "orchestrator", "", "Orchestrator where the App is running on (swarm, kubernetes)")
	cmd.Flags().StringVar(&opts.kubeNamespace, "namespace", "default", "Kubernetes namespace in which to find the App")
	opts.credentialOptions.addFlags(cmd.Flags())
	return cmd
}

func runInspect(dockerCli command.Cli, appName string, inspectOptions inspectOptions) error {
	orchestrator, err := getContextOrchestrator(dockerCli, inspectOptions.orchestrator)
	if err != nil {
		return err
	}
	services, err := stack.GetServices(dockerCli, pflag.NewFlagSet("", pflag.ContinueOnError), orchestrator, options.Services{
		Filter:    opts.NewFilterOpt(),
		Namespace: inspectOptions.kubeNamespace,
	})
	if err != nil {
		return err
	}
	println(services)

	inspectOptions.SetDefaultTargetContext(dockerCli)
	defer muteDockerCli(dockerCli)()
	_, installationStore, credentialStore, err := prepareStores(inspectOptions.targetContext)
	if err != nil {
		return err
	}
	installation, err := installationStore.Read(appName)
	if err != nil {
		return err
	}

	orchestratorName, ok := installation.Parameters[internal.ParameterOrchestratorName].(string)
	if !ok || orchestratorName == "" {
		orchestratorName = string(orchestrator)
	}

	format := "json"
	actionName := internal.ActionStatusJSONName
	if inspectOptions.pretty {
		format = "pretty"
		actionName = internal.ActionStatusName
	}

	if err := inspect.Inspect(os.Stdout, installation.Claim, format, orchestratorName); err != nil {
		return err
	}

	var statusAction bool
	for key := range installation.Bundle.Actions {
		if strings.HasPrefix(key, "io.cnab.status") {
			statusAction = true
		}
	}
	if !statusAction {
		return nil
	}

	bind, err := cnab.RequiredBindMount(inspectOptions.targetContext, orchestratorName, dockerCli.ContextStore())
	if err != nil {
		return err
	}

	driverImpl, errBuf := cnab.PrepareDriver(dockerCli, bind, nil)
	a := &action.RunCustom{
		Action: actionName,
		Driver: driverImpl,
	}

	creds, err := prepareCredentialSet(installation.Bundle, inspectOptions.CredentialSetOpts(dockerCli, credentialStore)...)
	if err != nil {
		return err
	}

	installation.SetParameter(internal.ParameterInspectFormatName, format)
	println()
	if err := a.Run(&installation.Claim, creds, nil); err != nil {
		return fmt.Errorf("inspect failed: %s\n%s", err, errBuf)
	}
	return nil
}

func getContextOrchestrator(dockerCli command.Cli, orchestratorFlag string) (command.Orchestrator, error) {
	var orchestrator command.Orchestrator
	orchestrator, _ = command.NormalizeOrchestrator(orchestratorFlag)
	if string(orchestrator) == "" {
		orchestrator, err := dockerCli.StackOrchestrator("")
		if err != nil {
			return "", err
		}
		return orchestrator, nil
	}
	return orchestrator, nil
}
