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
		return err
	}

	// iterate on contents
	for k, vs := range payload {
		v := []byte(vs)
		// Deal with windows/linux slashes
		convertedFilepath := filepath.FromSlash(k)

		// Check we aren't doing ./../../../ etc in the path
		fullFilepath := filepath.Join(appDir, convertedFilepath)
		_, err := filepath.Rel(appDir, fullFilepath)
		if err != nil {
			log.Warnf("dropping image entry '%s' with unexpected path outside of app dir", k)
			continue
		}
		if k == internal.MetadataFileName {
			log.Debug("Loading app metadata")
			v, err = updateMetadata(v, namespace, name, maintainers)
			if err != nil {
				return err
			}
		}

		// Create the directories for any nested files
		basepath := filepath.Dir(fullFilepath)
		os.MkdirAll(basepath, os.ModePerm)
		if err := ioutil.WriteFile(fullFilepath, v, 0644); err != nil {
			return errors.Wrapf(err, "failed to write output file: %s", fullFilepath)
		}
	}

	return nil
}

func updateMetadata(raw []byte, namespace, name string, maintainers []string) ([]byte, error) {
	// retrieve original metadata (maintainer/app name/app tag)
	var yamlMeta []byte
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
