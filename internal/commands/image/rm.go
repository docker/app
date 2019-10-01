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
		Use:   "rm [APP_IMAGE] [APP_IMAGE...]",
		Short: "Remove an application image",
		Args:  cli.RequiresMinArgs(1),
		Example: `$ docker app image rm myapp
$ docker app image rm myapp:1.0.0
$ docker app image rm docker.io/library/myapp@sha256:beef...`,
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
	ref, err := reference.ParseNormalizedNamed(app)
	if err != nil {
		return err
	}

	tagged := reference.TagNameOnly(ref)
	if err := bundleStore.Remove(tagged); err != nil {
		return err
	}

	fmt.Println("Deleted: " + reference.FamiliarString(tagged))
	return nil
}
