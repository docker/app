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

const inspectExample = `- $ docker app inspect my-running-app
- $ docker app inspect my-running-app:1.0.0`

type inspectOptions struct {
	credentialOptions
	pretty bool
}

func inspectCmd(dockerCli command.Cli, installerContext *cliopts.InstallerContextOptions) *cobra.Command {
	var opts inspectOptions
	cmd := &cobra.Command{
		Use:     "inspect [OPTIONS] RUNNING_APP",
		Short:   "Shows status, metadata, parameters and the list of services of a running App",
		Example: inspectExample,
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(dockerCli, args[0], opts, installerContext)
		},
	}
	cmd.Flags().BoolVar(&opts.pretty, "pretty", false, "Pretty print the output")
	opts.credentialOptions.addFlags(cmd.Flags())
	return cmd
}

func runInspect(dockerCli command.Cli, appName string, inspectOptions inspectOptions, installerContext *cliopts.InstallerContextOptions) error {
	defer muteDockerCli(dockerCli)()
	_, installationStore, credentialStore, err := prepareStores(dockerCli.CurrentContext())
	if err != nil {
		return err
	}
	installation, err := installationStore.Read(appName)
	if err != nil {
		return err
	}

	creds, err := prepareCredentialSet(installation.Bundle, inspectOptions.CredentialSetOpts(dockerCli, credentialStore)...)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	driverImpl, errBuf, err := cnab.SetupDriver(installation, dockerCli, installerContext, &buf)
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
	if err := a.Run(&installation.Claim, creds); err != nil {
		return fmt.Errorf("inspect failed: %s\n%s", err, errBuf)
	}

	if inspectOptions.pretty {
		if err := inspect.Inspect(os.Stdout, installation, "pretty"); err != nil {
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
			inspect.GetAppInfo(installation),
			statusJSON,
		}, "", "    ")
		if err != nil {
			return err
		}
		fmt.Fprint(os.Stdout, string(js))
	}
	return nil
}

func hasAction(bndl *bundle.Bundle, actionName string) bool {
	_, ok := bndl.Actions[actionName]
	return ok
}
