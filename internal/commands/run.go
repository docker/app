package commands

import (
	"fmt"
	"os"

	"github.com/docker/cli/cli"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/docker/app/internal/cnab"
	"github.com/docker/app/internal/store"
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
	cnabBundle    string
}

const longDescription = `Run an App from an App image.`

const example = `$ docker app run --name myrunningapp myrepo/myapp:mytag
$ docker app run 34be4a0c5f50 --name myrunningapp`

func runCmd(dockerCli command.Cli) *cobra.Command {
	var opts runOptions

	cmd := &cobra.Command{
		Use:     "run [OPTIONS] APP_IMAGE",
		Aliases: []string{"deploy"},
		Short:   "Run an App from an App image",
		Long:    longDescription,
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.cnabBundle != "" && len(args) != 0 {
				return errors.Errorf(
					"%q cannot run a bundle and an App image",
					cmd.CommandPath(),
				)
			}
			if opts.cnabBundle == "" {
				if err := cli.ExactArgs(1)(cmd, args); err != nil {
					return err
				}
				return runDockerApp(dockerCli, args[0], opts)
			}
			return runCnab(dockerCli, opts)
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
	opts.credentialOptions.addFlags(cmd.Flags())
	cmd.Flags().StringVar(&opts.orchestrator, "orchestrator", "", "Orchestrator to run on (swarm, kubernetes)")
	cmd.Flags().StringVar(&opts.kubeNamespace, "namespace", "default", "Kubernetes namespace in which to run the App")
	cmd.Flags().StringVar(&opts.stackName, "name", "", "Name of the running App")
	cmd.Flags().StringVar(&opts.cnabBundle, "cnab-bundle-json", "", "Run a CNAB bundle instead of a Docker App image")

	return cmd
}

func runCnab(dockerCli command.Cli, opts runOptions) error {
	bndl, err := cnab.LoadBundleFromFile(opts.cnabBundle)
	if err != nil {
		return errors.Wrapf(err, "failed to read bundle %q", opts.cnabBundle)
	}
	return runBundle(dockerCli, bndl, opts, "")
}

func runDockerApp(dockerCli command.Cli, appname string, opts runOptions) error {
	bundleStore, err := prepareBundleStore()
	if err != nil {
		return err
	}

	bndl, ref, err := cnab.GetBundle(dockerCli, bundleStore, appname)
	if err != nil {
		return errors.Wrapf(err, "Unable to find App %q", appname)
	}
	return runBundle(dockerCli, bndl, opts, ref.String())
}

func runBundle(dockerCli command.Cli, bndl *bundle.Bundle, opts runOptions, ref string) error {
	opts.SetDefaultTargetContext(dockerCli)

	bind, err := cnab.RequiredBindMount(opts.targetContext, opts.orchestrator, dockerCli.ContextStore())
	if err != nil {
		return err
	}
	_, installationStore, credentialStore, err := prepareStores(opts.targetContext)
	if err != nil {
		return err
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
			return fmt.Errorf("Installation %q already exists, use 'docker app update' instead", installationName)
		}
	} else {
		logrus.Debug(err)
	}
	installation, err := store.NewInstallation(installationName, ref)
	if err != nil {
		return err
	}

	driverImpl, errBuf := cnab.PrepareDriver(dockerCli, bind, nil)
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
		return fmt.Errorf("Failed to run App: %s\n%s", err, errBuf)
	}
	if err2 != nil {
		return err2
	}

	fmt.Fprintf(os.Stdout, "App %q is running on context %q\n", installationName, opts.targetContext)
	return nil
}
