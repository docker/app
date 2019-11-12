package image

import (
	"fmt"

	"github.com/docker/app/internal/relocated"

	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const tagExample = `- $ docker app image tag myapp myrepo/myapp:mytag
- $ docker app image tag myapp:tag myrepo/mynewapp:mytag
- $ docker app image tag 34be4a0c5f50 myrepo/mynewapp:mytag`

func tagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Short:   "Create a new tag from an App image",
		Use:     "tag SOURCE_APP_IMAGE[:TAG] TARGET_APP_IMAGE[:TAG]",
		Example: tagExample,
		Args:    cli.ExactArgs(2),
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

func readBundle(name string, bundleStore store.BundleStore) (*relocated.Bundle, error) {
	cnabRef, err := bundleStore.LookUp(name)
	if err != nil {
		switch err.(type) {
		case *store.UnknownReferenceError:
			return nil, fmt.Errorf("could not tag %q: no such App image", name)
		default:
			return nil, errors.Wrapf(err, "could not tag %q", name)
		}

	}

	bundle, err := bundleStore.Read(cnabRef)
	if err != nil {
		return nil, errors.Wrapf(err, "could not tag %q: no such App image", name)
	}
	return bundle, nil
}

func storeBundle(bundle *relocated.Bundle, name string, bundleStore store.BundleStore) error {
	cnabRef, err := store.StringToNamedRef(name)
	if err != nil {
		return err
	}
	_, err = bundleStore.Store(cnabRef, bundle)
	return err
}
