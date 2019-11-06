package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/cliopts"
	"github.com/docker/app/internal/cnab"
	"github.com/docker/app/internal/inspect"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type inspectOptions struct {
	credentialOptions
	cliopts.InstallerContextOptions
	pretty        bool
	orchestrator  string
	kubeNamespace string
}

func inspectCmd(dockerCli command.Cli) *cobra.Command {
	var opts inspectOptions
	cmd := &cobra.Command{
		Use:   "inspect [OPTIONS] RUNNING_APP",
		Short: "Shows installation and App metadata, parameters and the service list of a running App",
		Example: `$ docker app inspect my-running-app
$ docker app inspect my-running-app:1.0.0`,
		Args:   cli.ExactArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(dockerCli, firstOrEmpty(args), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.pretty, "pretty", false, "Pretty print the output")
	cmd.Flags().StringVar(&opts.orchestrator, "orchestrator", "", "Orchestrator where the App is running on (swarm, kubernetes)")
	cmd.Flags().StringVar(&opts.kubeNamespace, "namespace", "default", "Kubernetes namespace in which to find the App")
	opts.credentialOptions.addFlags(cmd.Flags())
	opts.InstallerContextOptions.AddFlags(cmd.Flags())
	return cmd
}

func runInspect(dockerCli command.Cli, appName string, inspectOptions inspectOptions) error {
	orchestrator, err := getContextOrchestrator(dockerCli, inspectOptions.orchestrator)
	if err != nil {
		return err
	}

	defer muteDockerCli(dockerCli)()
	_, installationStore, credentialStore, err := prepareStores(dockerCli.CurrentContext())
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

	creds, err := prepareCredentialSet(installation.Bundle, inspectOptions.CredentialSetOpts(dockerCli, credentialStore)...)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	driverImpl, errBuf, err := cnab.SetupDriver(installation, dockerCli, inspectOptions.InstallerContextOptions, &buf)
	if err != nil {
		return err
	}

	a := &action.RunCustom{
		Driver: driverImpl,
	}
	if inspectOptions.pretty && hasAction(installation.Bundle, internal.ActionStatusName) {
		a.Action = internal.ActionStatusName
	} else if hasAction(installation.Bundle, internal.ActionStatusJSONName) {
		a.Action = internal.ActionStatusJSONName
	} else {
		return fmt.Errorf("inspect failed: status action is not supported by the App")
	}
	if err := a.Run(&installation.Claim, creds, nil); err != nil {
		return fmt.Errorf("inspect failed: %s\n%s", err, errBuf)
	}

	if inspectOptions.pretty {
		if err := inspect.Inspect(os.Stdout, installation, "pretty", orchestratorName); err != nil {
			return err
		}
		fmt.Fprint(os.Stdout, buf.String())
	} else {
		var statusJSON interface{}
		if err := json.Unmarshal(buf.Bytes(), &statusJSON); err != nil {
			return err
		}
		js, err := json.MarshalIndent(struct {
			AppInfo  inspect.AppInfo `json:",omitempty"`
			Services interface{}     `json:",omitempty"`
		}{
			inspect.GetAppInfo(installation, orchestratorName),
			statusJSON,
		}, "", "    ")
		if err != nil {
			return err
		}
		fmt.Fprint(os.Stdout, string(js))
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

func hasAction(bndl *bundle.Bundle, actionName string) bool {
	for key := range bndl.Actions {
		if key == actionName {
			return true
		}
	}
	return false
}
