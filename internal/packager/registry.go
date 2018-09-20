package packager

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/pkg/resto"
	"github.com/docker/app/types"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type imageComponents struct {
	Name       string
	Repository string
	Tag        string
}

func splitImageName(repotag string) (*imageComponents, error) {
	named, err := reference.ParseNormalizedNamed(repotag)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse image name")
	}
	res := &imageComponents{
		Repository: named.Name(),
	}
	res.Name = res.Repository[strings.LastIndex(res.Repository, "/")+1:]
	if tagged, ok := named.(reference.Tagged); ok {
		res.Tag = tagged.Tag()
	}
	return res, nil
}

// Pull loads an app from a registry and returns the extracted dir name
func Pull(repotag string, outputDir string) (string, error) {
	payload, err := resto.PullConfigMulti(context.Background(), repotag, resto.RegistryOptions{})
	if err != nil {
		return "", err
	}
	repoComps, err := splitImageName(repotag)
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(outputDir, internal.DirNameFromAppName(repoComps.Name))
	err = os.Mkdir(appDir, 0755)
	if err != nil {
		return "", errors.Wrap(err, "failed to create output application directory")
	}
	for k, v := range payload {
		// do not write files in any other directory
		if strings.Contains(k, "/") || strings.Contains(k, "\\") {
			log.Warnf("dropping image entry '%s' with unexpected path separator", k)
			continue
		}
		target := filepath.Join(appDir, k)
		if err := ioutil.WriteFile(target, []byte(v), 0644); err != nil {
			return "", errors.Wrap(err, "failed to write output file")
		}
	}
	return appDir, nil
}

// Push pushes an app to a registry. Returns the image digest.
func Push(app *types.App, namespace, tag, repo string) (string, error) {
	payload, err := createPayload(app)
	if err != nil {
		return "", errors.Wrap(err, "failed to read external file whilst creating payload for push")
	}
	imageName := createImageName(app, namespace, tag, repo)
	return resto.PushConfigMulti(context.Background(), payload, imageName, resto.RegistryOptions{}, nil)
}

func createImageName(app *types.App, namespace, tag, repo string) string {
	if namespace == "" || tag == "" {
		metadata := app.Metadata()
		if namespace == "" {
			namespace = metadata.Namespace
		}
		if tag == "" {
			tag = metadata.Version
		}
	}
	if repo == "" {
		repo = internal.AppNameFromDir(app.Name) + internal.AppExtension
	}
	if namespace != "" && namespace[len(namespace)-1] != '/' {
		namespace += "/"
	}
	imageName := namespace + repo + ":" + tag
	return imageName
}

func createPayload(app *types.App) (map[string]string, error) {
	payload := map[string]string{
		internal.MetadataFileName: string(app.MetadataRaw()),
		internal.ComposeFileName:  string(app.Composes()[0]),
		internal.SettingsFileName: string(app.SettingsRaw()[0]),
	}
	err := readExternalFiles(payload, app.Path, app.ExternalFilePaths())

	return payload, err
}

func readExternalFiles(payload map[string]string, parentDirPath string, files []string) error {
	var errs []string
	for _, localfilepath := range files {
		fullFilePath := path.Join(parentDirPath, localfilepath)
		filedata, err := ioutil.ReadFile(fullFilePath)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		payload[localfilepath] = string(filedata)
	}
	return newErrGroup(errs)
}

func newErrGroup(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "\n"))
}
