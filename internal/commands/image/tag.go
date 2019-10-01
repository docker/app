package image

import (
	"fmt"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	"github.com/spf13/cobra"
)

func tagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Short: "Create a new tag from an application image",
		Use:   "tag SOURCE_APP_IMAGE[:TAG] TARGET_APP_IMAGE[:TAG]",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			appstore, err := store.NewApplicationStore(config.Dir())
			if err != nil {
				return err
			}

			bundleStore, err := appstore.BundleStore()
			if err != nil {
				return err
			}

			return runTag(bundleStore, args[0], args[1])
		},
	}

	return cmd
}

func runTag(bundleStore store.BundleStore, srcAppImage, destAppImage string) error {
	srcRef, err := readBundle(srcAppImage, bundleStore)
	if err != nil {
		return err
	}

	return storeBundle(srcRef, destAppImage, bundleStore)
}

func readBundle(name string, bundleStore store.BundleStore) (*bundle.Bundle, error) {
	cnabRef, err := stringToRef(name)
	if err != nil {
		return nil, err
	}

	return bundleStore.Read(cnabRef)
}

func storeBundle(bundle *bundle.Bundle, name string, bundleStore store.BundleStore) error {
	cnabRef, err := stringToRef(name)
	if err != nil {
		return err
	}

	return bundleStore.Store(cnabRef, bundle)
}

func stringToRef(name string) (reference.Named, error) {
	cnabRef, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return nil, fmt.Errorf("could not parse '%s' as a valid reference: %v", name, err)
	}

	return reference.TagNameOnly(cnabRef), nil
}
