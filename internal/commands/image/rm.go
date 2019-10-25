package image

import (
	"errors"
	"fmt"
	"strings"

	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	"github.com/spf13/cobra"
)

func rmCmd() *cobra.Command {
	return &cobra.Command{
		Short:   "Remove an App image",
		Use:     "rm APP_IMAGE [APP_IMAGE...]",
		Aliases: []string{"remove"},
		Args:    cli.RequiresMinArgs(1),
		Example: `$ docker app image rm myapp
$ docker app image rm myapp:1.0.0
$ docker app image rm myrepo/myapp@sha256:c0de...
$ docker app image rm 34be4a0c5f50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			appstore, err := store.NewApplicationStore(config.Dir())
			if err != nil {
				return err
			}

			bundleStore, err := appstore.BundleStore()
			if err != nil {
				return err
			}

			errs := []string{}
			for _, arg := range args {
				if err := runRm(bundleStore, arg); err != nil {
					errs = append(errs, fmt.Sprintf("Error: %s", err))
				}
			}
			if len(errs) > 0 {
				return errors.New(strings.Join(errs, "\n"))
			}
			return nil
		},
	}
}

func runRm(bundleStore store.BundleStore, app string) error {
	ref, err := bundleStore.LookUp(app)
	if err != nil {
		return err
	}

	if err := bundleStore.Remove(ref); err != nil {
		return err
	}

	fmt.Println("Deleted: " + reference.FamiliarString(ref))
	return nil
}
