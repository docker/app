package cnab

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/log"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/store"
	appstore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
)

type nameKind uint

const (
	_ nameKind = iota
	nameKindEmpty
	nameKindFile
	nameKindDir
	nameKindReference
)

func getAppNameKind(name string) (string, nameKind) {
	if name == "" {
		return name, nameKindEmpty
	}
	// name can be a bundle.json or bundle.cnab file, or a dockerapp directory
	st, err := os.Stat(name)
	if os.IsNotExist(err) {
		// try with .dockerapp extension
		st, err = os.Stat(name + internal.AppExtension)
		if err == nil {
			name += internal.AppExtension
		}
	}
	if err != nil {
		return name, nameKindReference
	}
	if st.IsDir() {
		return name, nameKindDir
	}
	return name, nameKindFile
}

func extractAndLoadAppBasedBundle(dockerCli command.Cli, name string) (*bundle.Bundle, string, error) {
	app, err := packager.Extract(name)
	if err != nil {
		return nil, "", err
	}
	defer app.Cleanup()
	bndl, err := packager.MakeBundleFromApp(dockerCli, app, nil)
	return bndl, "", err
}

func loadBundleFromFile(filename string) (*bundle.Bundle, error) {
	b := &bundle.Bundle{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return b, err
	}
	return bundle.Unmarshal(data)
}

// ResolveBundle looks for a CNAB bundle which can be in a Docker App Package format or
// a bundle stored locally or in the bundle store. It returns a built or found bundle,
// a reference to the bundle if it is found in the bundlestore, and an error.
func ResolveBundle(dockerCli command.Cli, bundleStore appstore.BundleStore, name string) (*bundle.Bundle, string, error) {
	// resolution logic:
	// - if there is a docker-app package in working directory, or an http:// / https:// prefix, use packager.Extract result
	// - the name has a .json or .cnab extension and refers to an existing file or web resource: load the bundle
	// - name matches a bundle name:version stored in the bundle store: use it
	// - pull the bundle from the registry and add it to the bundle store
	name, kind := getAppNameKind(name)
	switch kind {
	case nameKindFile:
		if strings.HasSuffix(name, internal.AppExtension) {
			return extractAndLoadAppBasedBundle(dockerCli, name)
		}
		bndl, err := loadBundleFromFile(name)
		return bndl, "", err
	case nameKindDir, nameKindEmpty:
		return extractAndLoadAppBasedBundle(dockerCli, name)
	case nameKindReference:
		bndl, tagRef, err := GetBundle(dockerCli, bundleStore, name)
		if err != nil {
			return nil, "", err
		}
		return bndl, tagRef.String(), err
	}
	return nil, "", fmt.Errorf("could not resolve bundle %q", name)
}

// GetBundle searches for the bundle locally and tries to pull it if not found
func GetBundle(dockerCli command.Cli, bundleStore appstore.BundleStore, name string) (*bundle.Bundle, reference.Reference, error) {
	ref, err := store.StringToRef(name)
	if err != nil {
		return nil, nil, err
	}
	bndl, err := bundleStore.Read(ref)
	if err != nil {
		fmt.Fprintf(dockerCli.Err(), "Unable to find application image %q locally\n", reference.FamiliarString(ref))

		fmt.Fprintf(dockerCli.Out(), "Pulling from registry...\n")
		if named, ok := ref.(reference.Named); ok {
			bndl, err = PullBundle(dockerCli, bundleStore, named)
			if err != nil {
				return nil, nil, err
			}
		}
	}

	return bndl, ref, nil
}

// PullBundle pulls the bundle and stores it into the bundle store
func PullBundle(dockerCli command.Cli, bundleStore appstore.BundleStore, tagRef reference.Named) (*bundle.Bundle, error) {
	insecureRegistries, err := internal.InsecureRegistriesFromEngine(dockerCli)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve insecure registries: %v", err)
	}

	bndl, err := remotes.Pull(log.WithLogContext(context.Background()), reference.TagNameOnly(tagRef), remotes.CreateResolver(dockerCli.ConfigFile(), insecureRegistries...))
	if err != nil {
		return nil, err
	}
	if _, err := bundleStore.Store(tagRef, bndl); err != nil {
		return nil, err
	}
	return bndl, nil
}
