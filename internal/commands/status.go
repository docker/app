package commands

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
)

var (
	knownStatusActions = []string{
		internal.ActionStatusName,
		// TODO: Extract this constant to the cnab-go library
		"io.cnab.status",
	}
)

func statusCmd(dockerCli command.Cli) *cobra.Command {
	var opts credentialOptions

	cmd := &cobra.Command{
		Use:     "status INSTALLATION_NAME [--target-context TARGET_CONTEXT] [OPTIONS]",
		Short:   "Get the installation status of an application",
		Long:    "Get the installation status of an application. If the installation is a Docker Application, the status shows the stack services.",
		Example: "$ docker app status myinstallation --target-context=mycontext",
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(dockerCli, args[0], opts)
		},
	}
	opts.addFlags(cmd.Flags())

	return cmd
}

func runStatus(dockerCli command.Cli, installationName string, opts credentialOptions) error {
	defer muteDockerCli(dockerCli)()
	opts.SetDefaultTargetContext(dockerCli)

	_, installationStore, credentialStore, err := prepareStores(opts.targetContext)
	if err != nil {
		return err
	}

	installation, err := installationStore.Read(installationName)
	if err != nil {
		return err
	}
	displayInstallationStatus(os.Stdout, installation)

	// Check if the bundle knows the docker app status action, if not just exit without error.
	statusAction := resolveStatusAction(installation)
	if statusAction == "" {
		return nil
	}

	bind, err := requiredClaimBindMount(installation.Claim, opts.targetContext, dockerCli)
	if err != nil {
		return err
	}
	driverImpl, errBuf, err := prepareDriver(dockerCli, bind, nil)
	if err != nil {
		return err
	}
	if err := mergeBundleParameters(installation,
		withSendRegistryAuth(opts.sendRegistryAuth),
	); err != nil {
		return err
	}
	creds, err := prepareCredentialSet(installation.Bundle, opts.CredentialSetOpts(dockerCli, credentialStore)...)
	if err != nil {
		return err
	}
	if err := credentials.Validate(creds, installation.Bundle.Credentials); err != nil {
		return err
	}
	printHeader(os.Stdout, "STATUS")
	status := &action.RunCustom{
		Action: statusAction,
		Driver: driverImpl,
	}
	if err := status.Run(&installation.Claim, creds, dockerCli.Out()); err != nil {
		return fmt.Errorf("status failed: %s\n%s", err, errBuf)
	}
	return nil
}

func displayInstallationStatus(w io.Writer, installation *store.Installation) {
	printHeader(w, "INSTALLATION")
	tab := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	printValue(tab, "Name", installation.Name)
	printValue(tab, "Created", units.HumanDuration(time.Since(installation.Created)))
	printValue(tab, "Modified", units.HumanDuration(time.Since(installation.Modified)))
	printValue(tab, "Revision", installation.Revision)
	printValue(tab, "Last Action", installation.Result.Action)
	printValue(tab, "Result", strings.ToUpper(installation.Result.Status))
	if o, ok := installation.Parameters[internal.ParameterOrchestratorName]; ok {
		orchestrator := fmt.Sprintf("%v", o)
		if orchestrator == "" {
			orchestrator = string(command.OrchestratorSwarm)
		}
		printValue(tab, "Orchestrator", orchestrator)
		if kubeNamespace, ok := installation.Parameters[internal.ParameterKubernetesNamespaceName]; ok && orchestrator == string(command.OrchestratorKubernetes) {
			printValue(tab, "Kubernetes namespace", fmt.Sprintf("%v", kubeNamespace))
		}
	}

	tab.Flush()
	fmt.Fprintln(w)

	printHeader(w, "APPLICATION")
	tab = tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	printValue(tab, "Name", installation.Bundle.Name)
	printValue(tab, "Version", installation.Bundle.Version)
	printValue(tab, "Reference", installation.Reference)
	tab.Flush()
	fmt.Fprintln(w)

	if len(installation.Parameters) > 0 {
		printHeader(w, "PARAMETERS")
		tab = tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
		params := sortParameters(installation)
		for _, param := range params {
			if !strings.HasPrefix(param, internal.Namespace) {
				// TODO: Trim long []byte parameters, maybe add type too (string, int...)
				printValue(tab, param, fmt.Sprintf("%v", installation.Parameters[param]))
			}
		}
		tab.Flush()
		fmt.Fprintln(w)
	}
}

func sortParameters(installation *store.Installation) []string {
	var params []string
	for name := range installation.Parameters {
		params = append(params, name)
	}
	sort.Strings(params)
	return params
}

func printHeader(w io.Writer, header string) {
	fmt.Fprintln(w, header)
	fmt.Fprintln(w, strings.Repeat("-", len(header)))
}

func printValue(w io.Writer, key, value string) {
	fmt.Fprintf(w, "%s:\t%s\n", key, value)
}

func resolveStatusAction(installation *store.Installation) string {
	for _, name := range knownStatusActions {
		if _, ok := installation.Bundle.Actions[name]; ok {
			return name
		}
	}
	return ""
}
