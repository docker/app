package packager

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/types"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
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
	if err := pullImage(originName); err != nil {
		return err
	}
	tmpdir, err := ioutil.TempDir("", "dockerappfork_")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)
	log.Debugf("Extracting original app data to %s", tmpdir)
	if err := Load(originName, tmpdir); err != nil {
		return err
	}

	// create app dir in output-dir
	namespace, name := splitPackageName(forkName)
	appPath := filepath.Join(outputDir, internal.DirNameFromAppName(name))
	os.MkdirAll(appPath, 0755)

	// iterate tar contents
	tarfile, err := os.Open(filepath.Join(tmpdir, internal.DirNameFromAppName(imgRef.Name)))
	if err != nil {
		return errors.Wrap(err, "failed to open package archive")
	}
	tarReader := tar.NewReader(tarfile)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		data := make([]byte, header.Size)
		if _, err := tarReader.Read(data); err != nil && err != io.EOF {
			return errors.Wrap(err, "error reading tar data")
		}

		if header.Name == internal.MetadataFileName {
			log.Debug("Loading app metadata")
			data, err = updateMetadata(data, namespace, name, maintainers)
			if err != nil {
				return err
			}
		}

		dest := filepath.Join(appPath, header.Name)
		log.Debugf("Writing file at %s", dest)
		if err := ioutil.WriteFile(dest, data, 0644); err != nil {
			return errors.Wrap(err, "error writing output file")
		}
	}

	return nil
}

func updateMetadata(raw []byte, namespace, name string, maintainers []string) ([]byte, error) {
	// retrieve original metadata (maintainer/app name/app tag)
	var yamlMeta []byte
	meta, err := loadMetadata(raw)
	if err != nil {
		return yamlMeta, err
	}
	// insert retrieved data in fork history section
	log.Debug("Generating fork metadata")
	newMeta := types.MetadataFrom(
		meta,
		types.WithName(name),
		types.WithNamespace(namespace),
		types.WithMaintainers(parseMaintainersData(maintainers)),
	)

	// update metadata file
	yamlMeta, err = yaml.Marshal(newMeta)
	if err != nil {
		return yamlMeta, errors.Wrap(err, "failed to render metadata structure")
	}
	return yamlMeta, nil
}

func loadMetadata(raw []byte) (types.AppMetadata, error) {
	var meta types.AppMetadata
	err := yaml.Unmarshal(raw, &meta)
	if err != nil {
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
