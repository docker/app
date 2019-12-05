package packager

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"github.com/docker/app/internal/image"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/store"
	"github.com/docker/app/types"
	"github.com/docker/app/types/metadata"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func MakeBundleFromApp(dockerCli command.Cli, app *types.App, refOverride reference.NamedTagged) (*bundle.Bundle, error) {
	logrus.Debug("Making app bundle")
	meta := app.Metadata()
	invocationImageName, err := MakeInvocationImageName(meta, refOverride)
	if err != nil {
		return nil, err
	}

	buildContext := bytes.NewBuffer(nil)
	if err := PackInvocationImageContext(dockerCli, app, buildContext); err != nil {
		return nil, err
	}

	logrus.Debugf("Building invocation image %s", invocationImageName)
	buildResp, err := dockerCli.Client().ImageBuild(context.TODO(), buildContext, dockertypes.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{invocationImageName},
		BuildArgs:  map[string]*string{},
	})
	if err != nil {
		return nil, err
	}
	defer buildResp.Body.Close()

	if err := jsonmessage.DisplayJSONMessagesStream(buildResp.Body, ioutil.Discard, 0, false, func(jsonmessage.JSONMessage) {}); err != nil {
		// If the invocation image can't be found we will get an error of the form:
		// manifest for docker/cnab-app-base:v0.6.0-202-gbaf0b246c7 not found
		if err.Error() == fmt.Sprintf("manifest for %s not found", BaseInvocationImage(dockerCli)) {
			return nil, fmt.Errorf("unable to resolve Docker App base image: %s", BaseInvocationImage(dockerCli))
		}
		return nil, err
	}

	return ToCNAB(app, invocationImageName)
}

func MakeInvocationImageName(meta metadata.AppMetadata, refOverride reference.NamedTagged) (string, error) {
	if refOverride != nil {
		return MakeCNABImageName(reference.FamiliarName(refOverride), refOverride.Tag(), "-invoc")
	}
	return MakeCNABImageName(meta.Name, meta.Version, "-invoc")
}

func MakeCNABImageName(appName, appVersion, suffix string) (string, error) {
	name := fmt.Sprintf("%s:%s%s", appName, appVersion, suffix)
	if _, err := reference.ParseNormalizedNamed(name); err != nil {
		return "", errors.Wrapf(err, "image name %q is invalid, please check name and version fields", name)
	}
	return name, nil
}

// PersistInImageStore do store a bundle with optional reference and return it's ID
func PersistInImageStore(ref reference.Reference, bndl *bundle.Bundle) (reference.Digested, error) {
	appstore, err := store.NewApplicationStore(config.Dir())
	if err != nil {
		return nil, err
	}
	imageStore, err := appstore.ImageStore()
	if err != nil {
		return nil, err
	}
	return imageStore.Store(image.FromBundle(bndl), ref)
}

func GetNamedTagged(tag string) (reference.NamedTagged, error) {
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
