package commands

import (
	"fmt"
	"os"

	"github.com/docker/app/internal/packager"

	"github.com/docker/app/internal/image"

	"github.com/cnabio/cnab-go/driver"
	"github.com/docker/app/internal/cliopts"

	"github.com/cnabio/cnab-go/action"
	"github.com/cnabio/cnab-go/credentials"
	bdl "github.com/docker/app/internal/bundle"
	"github.com/docker/app/internal/cnab"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type runOptions struct {
	cliopts.ParametersOptions
	credentialOptions
	orchestrator  string
	kubeNamespace string
	stackName     string
	cnabBundle    string
	labels        []string
}

const longDescription = `Run an App from an App image.`

const runExample = `- $ docker app run --name myrunningapp myrepo/myapp:mytag
- $ docker app run 34be4a0c5f50 --name myrunningapp`

func runCmd(dockerCli command.Cli, installerContext *cliopts.InstallerContextOptions) *cobra.Command {
	var opts runOptions

	cmd := &cobra.Command{
		Use:     "run [OPTIONS] APP_IMAGE",
		Aliases: []string{"deploy"},
		Short:   "Run an App from an App image",
		Long:    longDescription,
		Example: runExample,
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
				return runDockerApp(dockerCli, args[0], opts, installerContext)
			}
			return runCnab(dockerCli, opts, installerContext)
		},
	}
	opts.ParametersOptions.AddFlags(cmd.Flags())
	opts.credentialOptions.addFlags(cmd.Flags())
	cmd.Flags().StringVar(&opts.orchestrator, "orchestrator", "", "Orchestrator to install on (swarm, kubernetes)")
	cmd.Flags().StringVar(&opts.kubeNamespace, "namespace", "default", "Kubernetes namespace to install into")
	cmd.Flags().StringVar(&opts.stackName, "name", "", "Assign a name to the installation")
	cmd.Flags().StringVar(&opts.cnabBundle, "cnab-bundle-json", "", "Run a CNAB bundle instead of a Docker App")
	cmd.Flags().StringArrayVar(&opts.labels, "label", nil, "Label to add to services")

	//nolint:errcheck
	cmd.Flags().SetAnnotation("cnab-bundle-json", "experimentalCLI", []string{"true"})

	return cmd
}

func runCnab(dockerCli command.Cli, opts runOptions, installerContext *cliopts.InstallerContextOptions) error {
	bndl, err := image.FromFile(opts.cnabBundle)
	if err != nil {
		return errors.Wrapf(err, "failed to read bundle %q", opts.cnabBundle)
	}
	return runBundle(dockerCli, bndl, opts, installerContext, "")
}

func runDockerApp(dockerCli command.Cli, appname string, opts runOptions, installerContext *cliopts.InstallerContextOptions) error {
	imageStore, err := prepareImageStore()
	if err != nil {
		return err
	}

	bndl, ref, err := cnab.GetBundle(dockerCli, imageStore, appname)
	if err != nil {
		return errors.Wrapf(err, "Unable to find App %q", appname)
	}
	return runBundle(dockerCli, bndl, opts, installerContext, ref.String())
}

func runBundle(dockerCli command.Cli, bndl *image.AppImage, opts runOptions, installerContext *cliopts.InstallerContextOptions, ref string) (err error) {
	if err := packager.CheckAppVersion(dockerCli.Err(), bndl.Bundle); err != nil {
		return err
	}
	_, installationStore, credentialStore, err := prepareStores(dockerCli.CurrentContext())
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
		if IsInstallationFailed(installation) {
			fmt.Fprintf(dockerCli.Err(), "WARNING: installing over previously failed installation %q\n", installationName)
		} else {
			// Return an error in case of successful installation, or even failed upgrade, which means
			// their was already a successful installation.
			return fmt.Errorf("Installation %q already exists, use 'docker app update' instead", installationName)
		}
	} else {
		logrus.Debug(err)
	}
	installation, err := store.NewInstallation(installationName, ref, bndl)
	if err != nil {
		return err
	}

	driverImpl, errBuf, err := cnab.SetupDriver(installation, dockerCli, installerContext, os.Stdout)
	if err != nil {
		return err
	}

	if err := bdl.MergeBundleParameters(installation,
		bdl.WithFileParameters(opts.ParametersFiles),
		bdl.WithCommandLineParameters(opts.Overrides),
		bdl.WithLabels(opts.labels),
		bdl.WithOrchestratorParameters(opts.orchestrator, opts.kubeNamespace),
		bdl.WithSendRegistryAuth(opts.sendRegistryAuth),
	); err != nil {
		return err
	}
	creds, err := prepareCredentialSet(bndl.Bundle, opts.CredentialSetOpts(dockerCli, credentialStore)...)
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
		cfgFunc := func(op *driver.Operation) error {
			op.Out = dockerCli.Out()
			return nil
		}
		err = inst.Run(&installation.Claim, creds, cfgFunc, cnab.WithRelocationMap(installation))
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

	fmt.Fprintf(os.Stdout, "App %q running on context %q\n", installationName, dockerCli.CurrentContext())
	return nil
}
