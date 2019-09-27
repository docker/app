package commands

import (
	"fmt"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/store"
	"github.com/docker/app/types"
	"github.com/docker/app/types/metadata"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func makeBundleFromApp(dockerCli command.Cli, app *types.App, refOverride reference.NamedTagged) (*bundle.Bundle, error) {
	logrus.Debug("Making app bundle")
	meta := app.Metadata()
	invocationImageName, err := makeInvocationImageName(meta, refOverride)
	if err != nil {
		return nil, err
	}

	return packager.ToCNAB(app, invocationImageName)
}

func makeInvocationImageName(meta metadata.AppMetadata, refOverride reference.NamedTagged) (string, error) {
	if refOverride != nil {
		return makeCNABImageName(reference.FamiliarName(refOverride), refOverride.Tag(), "-invoc")
	}
	return makeCNABImageName(meta.Name, meta.Version, "-invoc")
}

func makeCNABImageName(appName, appVersion, suffix string) (string, error) {
	name := fmt.Sprintf("%s:%s%s", appName, appVersion, suffix)
	if _, err := reference.ParseNormalizedNamed(name); err != nil {
		return "", errors.Wrapf(err, "image name %q is invalid, please check name and version fields", name)
	}
	return name, nil
}

func persistInBundleStore(ref reference.Named, bndle *bundle.Bundle) error {
	if ref == nil {
		return nil
	}
	appstore, err := store.NewApplicationStore(config.Dir())
	if err != nil {
		return err
	}
	bundleStore, err := appstore.BundleStore()
	if err != nil {
		return err
	}
	return bundleStore.Store(ref, bndle)
}

func getNamedTagged(tag string) (reference.NamedTagged, error) {
	if tag == "" {
		return nil, nil
	}
	namedRef, err := reference.ParseNormalizedNamed(tag)
	if err != nil {
		return nil, err
	}
	ref, ok := reference.TagNameOnly(namedRef).(reference.NamedTagged)
	if !ok {
		return nil, fmt.Errorf("tag %q must be name with a tag in the 'name:tag' format", tag)
	}
	return ref, nil
}
