package packager

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/lunchbox/constants"
	"github.com/docker/lunchbox/types"
	"github.com/docker/lunchbox/utils"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

func appName(appname string) string {
	return utils.AppNameFromDir(appname)
}

// Save saves an app to docker and returns the image name.
func Save(appname, prefix, tag string) (string, error) {
	appname, cleanup, err := Extract(appname)
	if err != nil {
		return "", err
	}
	defer cleanup()
	if prefix == "" || tag == "" {
		metaFile := filepath.Join(appname, "metadata.yml")
		metaContent, err := ioutil.ReadFile(metaFile)
		if err != nil {
			return "", err
		}
		var meta types.AppMetadata
		err = yaml.Unmarshal(metaContent, &meta)
		if err != nil {
			return "", err
		}
		if tag == "" {
			tag = meta.Version
		}
		if prefix == "" {
			prefix = meta.RepositoryPrefix
		}
	}
	dockerfile := `
FROM scratch
COPY / /
`
	df := filepath.Join(appname, "__Dockerfile-docker-app__")
	ioutil.WriteFile(df, []byte(dockerfile), 0644)
	di := filepath.Join(appname, ".dockerignore")
	ioutil.WriteFile(di, []byte("__Dockerfile-docker-app__\n.dockerignore"), 0644)
	imageName := prefix + appName(appname) + constants.AppExtension + ":" + tag
	args := []string{"build", "-t", imageName, "-f", df, appname}
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	os.Remove(df)
	os.Remove(di)
	if err != nil {
		fmt.Println(string(output))
	}
	return imageName, err
}

// Load loads an app from docker
func Load(repotag string, outputDir string) error {
	file := filepath.Join(os.TempDir(), "docker-app-"+fmt.Sprintf("%v%v", rand.Int63(), rand.Int63()))
	cmd := exec.Command("docker", "save", "-o", file, repotag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "error from docker save command: %s", string(output))
	}
	defer os.Remove(file)
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(f)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "error reading next tar header")
		}
		if filepath.Base(header.Name) == "layer.tar" {
			data := make([]byte, header.Size)
			_, err := tarReader.Read(data)
			if err != nil && err != io.EOF {
				return errors.Wrap(err, "error reading tar data")
			}
			repo := strings.Split(repotag, ":")[0]
			err = ioutil.WriteFile(filepath.Join(outputDir, utils.DirNameFromAppName(filepath.Base(repo))), data, 0644)
			return errors.Wrap(err, "error writing output file")
		}
	}
	return fmt.Errorf("failed to find our layer in tarball")
}

// Push pushes an app to a registry
func Push(appname, prefix, tag string) error {
	appname, cleanup, err := Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
	imageName, err := Save(appname, prefix, tag)
	if err != nil {
		return err
	}
	cmd := exec.Command("docker", "push", imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "error from docker push command: %s", string(output))
	}
	return nil
}

// Pull pulls an app from a registry
func Pull(repotag string) error {
	cmd := exec.Command("docker", "pull", repotag)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "error from docker pull command: %s", string(output))
	}
	return Load(repotag, ".")
}
