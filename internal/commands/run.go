package commands

import (
	"fmt"
	"os"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type runOptions struct {
	parametersOptions
	credentialOptions
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

const longDescription = `Run an application.
By default, the application definition in the current directory will be ran. The APP_NAME can also be:
- a path to a Docker Application definition (.dockerapp) or a CNAB bundle.json
- a registry Application Package reference`

const example = `$ docker app run myapp.dockerapp --name myinstallation --target-context=mycontext
$ docker app run myrepo/myapp:mytag --name myinstallation --target-context=mycontext
$ docker app run bundle.json --name myinstallation --credential-set=mycredentials.yml`

func runCmd(dockerCli command.Cli) *cobra.Command {
	var opts runOptions

	cmd := &cobra.Command{
		Use:     "run [APP_NAME] [--name INSTALLATION_NAME] [--target-context TARGET_CONTEXT] [OPTIONS]",
		Aliases: []string{"deploy"},
		Short:   "Run an application",
		Long:    longDescription,
		Example: example,
		Args:    cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRun(dockerCli, firstOrEmpty(args), opts)
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
	opts.credentialOptions.addFlags(cmd.Flags())
	cmd.Flags().StringVar(&opts.orchestrator, "orchestrator", "", "Orchestrator to install on (swarm, kubernetes)")
	cmd.Flags().StringVar(&opts.kubeNamespace, "namespace", "default", "Kubernetes namespace to install into")
	cmd.Flags().StringVar(&opts.stackName, "name", "", "Assign a name to the installation")

	return cmd
}

func runRun(dockerCli command.Cli, appname string, opts runOptions) error {
	opts.SetDefaultTargetContext(dockerCli)

	bind, err := requiredBindMount(opts.targetContext, opts.orchestrator, dockerCli.ContextStore())
	if err != nil {
		return err
	}
	bundleStore, installationStore, credentialStore, err := prepareStores(opts.targetContext)
	if err != nil {
		return err
	}

	bndl, ref, err := resolveBundle(dockerCli, bundleStore, appname)
	if err != nil {
		return errors.Wrapf(err, "Unable to find application %q", appname)
	}
	if err := bndl.Validate(); err != nil {
		return err
	}
	installationName := opts.stackName
	if installationName == "" {
		installationName = namesgenerator.GetRandomName(0)
	}
	logrus.Debugf(`Looking for a previous installation "%q"`, installationName)
	if installation, err := installationStore.Read(installationName); err == nil {
		// A failed installation can be overridden, but with a warning
		if isInstallationFailed(installation) {
			fmt.Fprintf(os.Stderr, "WARNING: installing over previously failed installation %q\n", installationName)
		} else {
			// Return an error in case of successful installation, or even failed upgrade, which means
			// their was already a successful installation.
			return fmt.Errorf("Installation %q already exists, use 'docker app upgrade' instead", installationName)
		}
	} else {
		logrus.Debug(err)
	}
	installation, err := store.NewInstallation(installationName, ref)
	if err != nil {
		return err
	}

	driverImpl, errBuf := prepareDriver(dockerCli, bind, nil)
	installation.Bundle = bndl

	if err := mergeBundleParameters(installation,
		withFileParameters(opts.parametersFiles),
		withCommandLineParameters(opts.overrides),
		withOrchestratorParameters(opts.orchestrator, opts.kubeNamespace),
		withSendRegistryAuth(opts.sendRegistryAuth),
	); err != nil {
		return err
	}
	creds, err := prepareCredentialSet(bndl, opts.CredentialSetOpts(dockerCli, credentialStore)...)
	if err != nil {
		return err
	}
	if err := credentials.Validate(creds, bndl.Credentials); err != nil {
		return err
	}

	inst := &action.Install{
		Driver: driverImpl,
	}
	{
		defer muteDockerCli(dockerCli)()
		err = inst.Run(&installation.Claim, creds, os.Stdout)
	}
	// Even if the installation failed, the installation is persisted with its failure status,
	// so any installation needs a clean uninstallation.
	err2 := installationStore.Store(installation)
	if err != nil {
		return fmt.Errorf("Installation failed: %s\n%s", err, errBuf)
	}
	if err2 != nil {
		return err2
	}

	fmt.Fprintf(os.Stdout, "Application %q installed on context %q\n", installationName, opts.targetContext)
	return nil
}
