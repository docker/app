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

	"github.com/docker/app/constants"
	"github.com/docker/app/types"
	"github.com/docker/app/utils"
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
	metaFile := filepath.Join(appname, "metadata.yml")
	metaContent, err := ioutil.ReadFile(metaFile)
	if err != nil {
		return "", errors.Wrap(err, "failed to read application metadata")
	}
	var meta types.AppMetadata
	err = yaml.Unmarshal(metaContent, &meta)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse application metadata")
	}
	if tag == "" {
		tag = meta.Version
	}
	if prefix == "" {
		prefix = meta.RepositoryPrefix
	}
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	dockerfile := fmt.Sprintf(`
FROM scratch
LABEL com.docker.application=%s
LABEL maintainers="%v"
COPY / /
`, meta.Name, meta.Maintainers)
	df := filepath.Join(appname, "__Dockerfile-docker-app__")
	if err := ioutil.WriteFile(df, []byte(dockerfile), 0644); err != nil {
		return "", errors.Wrapf(err, "cannot create file %s", df)
	}
	defer os.Remove(df)
	di := filepath.Join(appname, ".dockerignore")
	if err := ioutil.WriteFile(di, []byte("__Dockerfile-docker-app__\n.dockerignore"), 0644); err != nil {
		return "", errors.Wrapf(err, "cannot create file %s", di)
	}
	defer os.Remove(di)
	imageName := prefix + appName(appname) + constants.AppExtension + ":" + tag
	args := []string{"build", "-t", imageName, "-f", df, appname}
	cmd := exec.Command("docker", args...)
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	return imageName, err
}

// Load loads an app from docker
func Load(repotag string, outputDir string) error {
	file := filepath.Join(os.TempDir(), "docker-app-"+fmt.Sprintf("%v%v", rand.Int63(), rand.Int63()))
	cmd := exec.Command("docker", "save", "-o", file, repotag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "error loading image %s", repotag)
	}
	defer os.Remove(file)
	f, err := os.Open(file)
	if err != nil {
		return errors.Wrap(err, "failed to open temporary image file")
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
			repoComps := strings.Split(repotag, ":")
			repo := repoComps[0]
			if len(repoComps) == 3 || (len(repoComps) == 2 && strings.Contains(repoComps[1], "/")) {
				repo = repoComps[1]
			}
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return errors.Wrapf(cmd.Run(), "error pushing image %s", imageName)
}

// Pull pulls an app from a registry
func Pull(repotag string) error {
	cmd := exec.Command("docker", "pull", repotag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "error pulling image %s", repotag)
	}
	return Load(repotag, ".")
}
