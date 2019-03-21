package commands

import (
	"fmt"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/utils/crud"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type installOptions struct {
	parametersOptions
	credentialOptions
	registryOptions
	pullOptions
	orchestrator  string
	kubeNamespace string
	stackName     string
}

type nameKind uint

const (
	_ nameKind = iota
	nameKindEmpty
	nameKindFile
	nameKindDir
	nameKindReference
)

const longDescription = `Install the application on either Swarm or Kubernetes.
Bundle name is optional, and can:
- be empty and resolve to any *.dockerapp in working directory
- be a BUNDLE file path and resolve to any *.dockerapp file or dir, or any CNAB file (signed or unsigned)
- match a bundle name in the local duffle bundle repository
- refer to a CNAB in a container registry
`

func installCmd(dockerCli command.Cli) *cobra.Command {
	var opts installOptions

	cmd := &cobra.Command{
		Use:     "install [<bundle name>] [OPTIONS]",
		Aliases: []string{"deploy"},
		Short:   "Install an application",
		Long:    longDescription,
		Args:    cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(dockerCli, firstOrEmpty(args), opts)
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
	opts.credentialOptions.addFlags(cmd.Flags())
	opts.registryOptions.addFlags(cmd.Flags())
	opts.pullOptions.addFlags(cmd.Flags())
	cmd.Flags().StringVarP(&opts.orchestrator, "orchestrator", "o", "", "Orchestrator to install on (swarm, kubernetes)")
	cmd.Flags().StringVar(&opts.kubeNamespace, "kubernetes-namespace", "default", "Kubernetes namespace to install into")
	cmd.Flags().StringVar(&opts.stackName, "name", "", "Installation name (defaults to application name)")

	return cmd
}

func runInstall(dockerCli command.Cli, appname string, opts installOptions) error {
	defer muteDockerCli(dockerCli)()
	targetContext := getTargetContext(opts.targetContext, dockerCli.CurrentContext())
	bind, err := requiredBindMount(targetContext, opts.orchestrator, dockerCli.ContextStore())
	if err != nil {
		return err
	}
	bndl, err := resolveBundle(dockerCli, appname, opts.pull, opts.insecureRegistries)
	if err != nil {
		return err
	}
	if err := bndl.Validate(); err != nil {
		return err
	}
	h := duffleHome()
	claimName := opts.stackName
	if claimName == "" {
		claimName = bndl.Name
	}
	claimStore := claim.NewClaimStore(crud.NewFileSystemStore(h.Claims(), "json"))
	if _, err = claimStore.Read(claimName); err == nil {
		return fmt.Errorf("installation %q already exists", claimName)
	}
	c, err := claim.New(claimName)
	if err != nil {
		return err
	}

	driverImpl, errBuf, err := prepareDriver(dockerCli, bind, nil)
	if err != nil {
		return err
	}
	c.Bundle = bndl

	if err := mergeBundleParameters(c,
		withFileParameters(opts.parametersFiles),
		withCommandLineParameters(opts.overrides),
		withOrchestratorParameters(opts.orchestrator, opts.kubeNamespace),
		withSendRegistryAuth(opts.sendRegistryAuth),
	); err != nil {
		return err
	}
	creds, err := prepareCredentialSet(bndl,
		addNamedCredentialSets(opts.credentialsets),
		addDockerCredentials(targetContext, dockerCli.ContextStore()),
		addRegistryCredentials(opts.sendRegistryAuth, dockerCli))
	if err != nil {
		return err
	}
	if err := credentials.Validate(creds, bndl.Credentials); err != nil {
		return err
	}

	inst := &action.Install{
		Driver: driverImpl,
	}
	err = inst.Run(c, creds, dockerCli.Out())
	// Even if the installation failed, the claim is persisted with its failure status,
	// so any installation needs a clean uninstallation.
	err2 := claimStore.Store(*c)
	if err != nil {
		return fmt.Errorf("install failed: %s", errBuf)
	}
	return err2
}
