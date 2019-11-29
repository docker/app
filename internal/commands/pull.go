package commands

import (
	"fmt"
	"os"

	"github.com/docker/app/internal/cnab"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func pullCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pull APP_IMAGE",
		Short:   "Pull an App image from a registry",
		Example: `$ docker app pull myrepo/myapp:0.1.0`,
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPull(dockerCli, args[0])
		},
	}
	return cmd
}

func runPull(dockerCli command.Cli, name string) error {
	appstore, err := store.NewApplicationStore(config.Dir())
	if err != nil {
		return err
	}
	bundleStore, err := appstore.BundleStore()
	if err != nil {
		return err
	}
	ref, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return errors.Wrap(err, name)
	}
	tagRef := reference.TagNameOnly(ref)

	bndl, err := cnab.PullBundle(dockerCli, bundleStore, tagRef)
	if err != nil {
		return errors.Wrap(err, name)
	}
	if err := packager.CheckAppVersion(dockerCli.Err(), bndl.Bundle); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "Successfully pulled %q (%s) from %s\n", bndl.Name, bndl.Version, ref.String())

	return nil
}
