package packager

import (
	"context"
	"io/ioutil"
	"os"
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
	imgRef, err := splitImageName(repotag)
	if err != nil {
		return "", errors.Wrapf(err, "origin %q is not a valid image name", repotag)
	}
	payload, err := resto.PullConfigMulti(context.Background(), repotag, resto.RegistryOptions{})
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(outputDir, internal.DirNameFromAppName(imgRef.Name))
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", errors.Wrap(err, "failed to create output application directory")
	}
	if err := ExtractImagePayloadToDiskFiles(appDir, payload); err != nil {
		return "", err
	}
	return appDir, nil
}

// ExtractImagePayloadToDiskFiles extracts all the files out of the image payload and onto disk
// creating all necessary folders in between.
func ExtractImagePayloadToDiskFiles(appDir string, payload map[string]string) error {
	for localFilepath, filedata := range payload {
		fileBytes := []byte(filedata)
		// Deal with windows/linux slashes
		convertedFilepath := filepath.FromSlash(localFilepath)

		// Check we aren't doing ./../../../ etc in the path
		fullFilepath := filepath.Join(appDir, convertedFilepath)
		_, err := filepath.Rel(appDir, fullFilepath)
		if err != nil {
			log.Warnf("dropping image entry '%s' with unexpected path outside of app dir", localFilepath)
			continue
		}

		// Create the directories for any nested files
		basepath := filepath.Dir(fullFilepath)
		if err := os.MkdirAll(basepath, os.ModePerm); err != nil {
			return errors.Wrapf(err, "failed to create directories for file: %s", fullFilepath)
		}
		if err := ioutil.WriteFile(fullFilepath, fileBytes, 0644); err != nil {
			return errors.Wrapf(err, "failed to write output file: %s", fullFilepath)
		}
	}

	return nil
}

// Push pushes an app to a registry. Returns the image digest.
func Push(app *types.App, namespace, tag, repo string) (string, error) {
	payload, err := createPayload(app)
	if err != nil {
		return "", errors.Wrap(err, "failed to read external file while creating payload for push")
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
	return namespace + repo + ":" + tag
}

func createPayload(app *types.App) (map[string]string, error) {
	payload := map[string]string{
		internal.MetadataFileName: string(app.MetadataRaw()),
		internal.ComposeFileName:  string(app.Composes()[0]),
		internal.SettingsFileName: string(app.SettingsRaw()[0]),
	}
	if err := readAttachments(payload, app.Path, app.Attachments()); err != nil {
		return nil, err
	}
	return payload, nil
}

func readAttachments(payload map[string]string, parentDirPath string, files []types.Attachment) error {
	var errs []string
	for _, file := range files {
		// Convert to local OS filepath slash syntax
		fullFilePath := filepath.Join(parentDirPath, filepath.FromSlash(file.FilePath()))
		filedata, err := ioutil.ReadFile(fullFilePath)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		payload[file.FilePath()] = string(filedata)
	}
	return newErrGroup(errs)
}

func newErrGroup(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "\n"))
}
