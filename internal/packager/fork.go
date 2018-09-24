package packager

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/yaml"
	"github.com/docker/app/pkg/resto"
	"github.com/docker/app/types/metadata"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// Fork pulls an application and creates a local fork for the user to modify
func Fork(originName, forkName, outputDir string, maintainers []string) error {
	imgRef, err := splitImageName(originName)
	if err != nil {
		return errors.Wrapf(err, "origin %q is not a valid image name", originName)
	}
	if forkName == "" {
		forkName = internal.AppNameFromDir(imgRef.Name)
	}
	log.Debugf("Pulling latest version of package %s", originName)
	payload, err := resto.PullConfigMulti(context.Background(), originName, resto.RegistryOptions{})
	if err != nil {
		return err
	}

	// create app dir in output-dir
	namespace, name := splitPackageName(forkName)
	appDir := filepath.Join(outputDir, internal.DirNameFromAppName(name))
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create output application directory")
	}

	// Create all the files on disk
	if err := ExtractImagePayloadToDiskFiles(appDir, payload); err != nil {
		return err
	}

	// Update the metadata file
	fullFilepath := filepath.Join(appDir, internal.MetadataFileName)
	bytes, err := ioutil.ReadFile(fullFilepath)
	if err != nil {
		return errors.Wrapf(err, "failed to read metadata file from: %s", fullFilepath)
	}
	log.Debug("Loading app metadata")
	bytes, err = updateMetadata(bytes, namespace, name, maintainers)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(fullFilepath, bytes, 0644); err != nil {
		return errors.Wrapf(err, "failed to write metadata file: %s", fullFilepath)
	}

	return nil
}

func updateMetadata(raw []byte, namespace, name string, maintainers []string) ([]byte, error) {
	// retrieve original metadata (maintainer/app name/app tag)
	meta, err := loadMetadata(raw)
	if err != nil {
		return nil, err
	}
	// insert retrieved data in fork history section
	log.Debug("Generating fork metadata")
	newMeta := metadata.From(
		meta,
		metadata.WithName(name),
		metadata.WithNamespace(namespace),
		metadata.WithMaintainers(parseMaintainersData(maintainers)),
	)

	// update metadata file
	var yamlMeta []byte
	yamlMeta, err = yaml.Marshal(newMeta)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render metadata structure")
	}
	return yamlMeta, nil
}

func loadMetadata(raw []byte) (metadata.AppMetadata, error) {
	var meta metadata.AppMetadata
	if err := yaml.Unmarshal(raw, &meta); err != nil {
		return meta, errors.Wrap(err, "failed to parse application metadata")
	}
	return meta, nil
}

func splitPackageName(name string) (string, string) {
	ls := strings.LastIndexByte(name, '/')
	if ls == -1 {
		return "", name
	}

	return name[:ls], name[ls+1:]
}
