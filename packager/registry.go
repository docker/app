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
	"github.com/docker/lunchbox/utils"
	"github.com/pkg/errors"
)

func appName(appname string) string {
	return utils.AppNameFromDir(appname)
}

// Save saves an app to docker
func Save(appname, prefix, tag string) error {
	appname, cleanup, err := Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
	dockerfile := `
FROM scratch
COPY / /
`
	df := filepath.Join(appname, "__Dockerfile-docker-app__")
	ioutil.WriteFile(df, []byte(dockerfile), 0644)
	di := filepath.Join(appname, ".dockerignore")
	ioutil.WriteFile(di, []byte("__Dockerfile-docker-app__\n.dockerignore"), 0644)
	args := []string{"build", "-t", prefix + appName(appname) + constants.AppExtension + ":" + tag, "-f", df, appname}
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	os.Remove(df)
	os.Remove(di)
	if err != nil {
		fmt.Println(string(output))
	}
	return err
}

// Load loads an app from docker
func Load(repotag string) error {
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
			err = ioutil.WriteFile(appName(repo)+constants.AppExtension, data, 0644)
			return errors.Wrap(err, "error writing output file")
		}
	}
	return fmt.Errorf("failed to find our layer in tarball")
}

// Push pushes an app to a registry
func Push(appname, prefix, tag string) error {
	err := Save(appname, prefix, tag)
	if err != nil {
		return err
	}
	cmd := exec.Command("docker", "push", prefix+appName(appname)+constants.AppExtension+":"+tag)
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
	return Load(repotag)
}
