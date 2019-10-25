package image

import (
	"fmt"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/config"
	"github.com/spf13/cobra"
)

func tagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Short: "Create a new tag from an App image",
		Use:   "tag SOURCE_APP_IMAGE[:TAG] TARGET_APP_IMAGE[:TAG]",
		Example: `$ docker app image tag myapp myrepo/myapp:mytag
$ docker app image tag myapp:tag myrepo/mynewapp:mytag
$ docker app image tag 34be4a0c5f50 myrepo/mynewapp:mytag`,
		Args: cli.ExactArgs(2),
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
	cnabRef, err := store.StringToRef(name)
	if err != nil {
		return nil, err
	}

	bundle, err := bundleStore.Read(cnabRef)
	if err != nil {
		return nil, fmt.Errorf("could not tag '%s': no such App image", name)
	}
	return bundle, nil
}

func storeBundle(bundle *bundle.Bundle, name string, bundleStore store.BundleStore) error {
	cnabRef, err := store.StringToRef(name)
	if err != nil {
		return err
	}
	_, err = bundleStore.Store(cnabRef, bundle)
	return err
}
