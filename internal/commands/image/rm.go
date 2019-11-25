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

const rmExample = `- $ docker app image rm myapp
- $ docker app image rm myapp:1.0.0
- $ docker app image rm myrepo/myapp@sha256:c0de...
- $ docker app image rm 34be4a0c5f50
- $ docker app image rm --force 34be4a0c5f50`

type rmOptions struct {
	force bool
}

func rmCmd() *cobra.Command {
	options := rmOptions{}
	cmd := &cobra.Command{
		Short:   "Remove an App image",
		Use:     "rm APP_IMAGE [APP_IMAGE...]",
		Aliases: []string{"remove"},
		Args:    cli.RequiresMinArgs(1),
		Example: rmExample,
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
				if err := runRm(bundleStore, arg, options); err != nil {
					errs = append(errs, fmt.Sprintf("Error: %s", err))
				}
			}
			if len(errs) > 0 {
				return errors.New(strings.Join(errs, "\n"))
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&options.force, "force", "f", false, "")
	return cmd
}

func runRm(bundleStore store.BundleStore, app string, options rmOptions) error {
	ref, err := bundleStore.LookUp(app)
	if err != nil {
		return err
	}

	if err := bundleStore.Remove(ref, options.force); err != nil {
		return err
	}

	fmt.Println("Deleted: " + reference.FamiliarString(ref))
	return nil
}
