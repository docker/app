package main

import (
	"fmt"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/utils/crud"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type installOptions struct {
	parametersOptions
	credentialOptions
	orchestrator     string
	namespace        string
	kubeNamespace    string
	stackName        string
	insecure         bool
	sendRegistryAuth bool
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
	cmd.Flags().StringVarP(&opts.orchestrator, "orchestrator", "o", "", "Orchestrator to install on (swarm, kubernetes)")
	cmd.Flags().StringVar(&opts.namespace, "namespace", "", "Namespace to use (default: namespace in metadata)")
	cmd.Flags().StringVar(&opts.kubeNamespace, "kubernetes-namespace", "default", "Kubernetes namespace to install into")
	cmd.Flags().StringVar(&opts.stackName, "name", "", "Installation name (defaults to application name)")
	cmd.Flags().BoolVar(&opts.insecure, "insecure", false, "Use insecure registry, without SSL")
	cmd.Flags().BoolVar(&opts.sendRegistryAuth, "with-registry-auth", false, "Sends registry auth")

	return cmd
}

func runInstall(dockerCli command.Cli, appname string, opts installOptions) error {
	muteDockerCli(dockerCli)
	targetContext := getTargetContext(opts.targetContext, dockerCli.CurrentContext())
	parameterValues, err := prepareParameters(opts.parametersOptions)
	if err != nil {
		return err
	}

	bndl, err := resolveBundle(dockerCli, opts.namespace, appname)
	if err != nil {
		return err
	}
	if opts.sendRegistryAuth {
		return errors.New("with-registry-auth is not supported at the moment")
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
	if _, ok := bndl.Parameters["docker.orchestrator"]; ok {
		parameterValues["docker.orchestrator"] = opts.orchestrator
	}
	if _, ok := bndl.Parameters["docker.kubernetes-namespace"]; ok {
		parameterValues["docker.kubernetes-namespace"] = opts.kubeNamespace
	}

	driverImpl, err := prepareDriver(dockerCli)
	if err != nil {
		return err
	}
	creds, err := prepareCredentialSet(targetContext, dockerCli.ContextStore(), bndl, opts.credentialsets)
	if err != nil {
		return err
	}
	if err := credentials.Validate(creds, bndl.Credentials); err != nil {
		return err
	}

	c.Bundle = bndl
	convertedParamValues := map[string]interface{}{}

	if err := applyParameterValues(parameterValues, bndl.Parameters, convertedParamValues); err != nil {
		return err
	}

	c.Parameters, err = bundle.ValuesOrDefaults(convertedParamValues, bndl)
	if err != nil {
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
		return fmt.Errorf("install failed: %v", err)
	}
	return err2
}
