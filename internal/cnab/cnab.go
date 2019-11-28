package cnab

import (
	"context"
	"fmt"
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/image"
	"github.com/docker/app/internal/log"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/store"
	appstore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
	"github.com/sirupsen/logrus"
)

type nameKind uint

const (
	_ nameKind = iota
	nameKindDir
	nameKindReference
)

func getAppNameKind(name string) (string, nameKind) {
	if name == "" {
		return name, nameKindDir
	}
	// name can be a dockerapp directory
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
	return name, nameKindReference
}

func extractAndLoadAppBasedBundle(dockerCli command.Cli, name string) (*image.AppImage, string, error) {
	app, err := packager.Extract(name)
	if err != nil {
		return nil, "", err
	}
	defer app.Cleanup()
	bndl, err := packager.MakeBundleFromApp(dockerCli, app, nil)
	return image.FromBundle(bndl), "", err
}

// ResolveBundle looks for a CNAB bundle which can be in a Docker App Package format or
// a bundle stored locally or in the bundle store. It returns a built or found bundle,
// a reference to the bundle if it is found in the imageStore, and an error.
func ResolveBundle(dockerCli command.Cli, imageStore appstore.ImageStore, name string) (*image.AppImage, string, error) {
	// resolution logic:
	// - if there is a docker-app package in working directory or if a directory is given use packager.Extract
	// - pull the bundle from the registry and add it to the bundle store
	name, kind := getAppNameKind(name)
	switch kind {
	case nameKindDir:
		return extractAndLoadAppBasedBundle(dockerCli, name)
	case nameKindReference:
		bndl, tagRef, err := GetBundle(dockerCli, imageStore, name)
		if err != nil {
			return nil, "", err
		}
		return bndl, tagRef.String(), err
	}
	return nil, "", fmt.Errorf("could not resolve bundle %q", name)
}

// GetBundle searches for the bundle locally and tries to pull it if not found
func GetBundle(dockerCli command.Cli, imageStore appstore.ImageStore, name string) (*image.AppImage, reference.Reference, error) {
	bndl, ref, err := getBundleFromStore(imageStore, name)
	if err != nil {
		named, err := store.StringToNamedRef(name)
		if err != nil {
			return nil, nil, err
		}
		fmt.Fprintf(dockerCli.Err(), "Unable to find App image %q locally\n", reference.FamiliarString(named))
		fmt.Fprintf(dockerCli.Out(), "Pulling from registry...\n")
		bndl, err = PullBundle(dockerCli, imageStore, named)
		if err != nil {
			return nil, nil, err
		}
		ref = named
	}
	return bndl, ref, nil
}

func getBundleFromStore(imageStore appstore.ImageStore, name string) (*image.AppImage, reference.Reference, error) {
	ref, err := imageStore.LookUp(name)
	if err != nil {
		logrus.Debugf("Unable to find reference %q in the bundle store", name)
		return nil, nil, err
	}
	bndl, err := imageStore.Read(ref)
	if err != nil {
		logrus.Debugf("Unable to read bundle %q from store", reference.FamiliarString(ref))
		return nil, nil, err
	}
	return bndl, ref, nil
}

// PullBundle pulls the bundle and stores it into the bundle store
func PullBundle(dockerCli command.Cli, imageStore appstore.ImageStore, tagRef reference.Named) (*image.AppImage, error) {
	insecureRegistries, err := internal.InsecureRegistriesFromEngine(dockerCli)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve insecure registries: %v", err)
	}

	bndl, relocationMap, err := remotes.Pull(log.WithLogContext(context.Background()), reference.TagNameOnly(tagRef), remotes.CreateResolver(dockerCli.ConfigFile(), insecureRegistries...))
	if err != nil {
		return nil, err
	}
	relocatedBundle := &image.AppImage{Bundle: bndl, RelocationMap: relocationMap}
	if _, err := imageStore.Store(tagRef, relocatedBundle); err != nil {
		return nil, err
	}
	return relocatedBundle, nil
}
