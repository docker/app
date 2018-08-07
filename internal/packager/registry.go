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

	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Save saves an app to docker and returns the image name.
func Save(app *types.App, namespace, tag string) (string, error) {
	var meta types.AppMetadata
	err := yaml.Unmarshal(app.Metadata(), &meta)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse application metadata")
	}
	if tag == "" {
		tag = meta.Version
	}
	if namespace == "" {
		namespace = meta.Namespace
	}
	if namespace != "" && !strings.HasSuffix(namespace, "/") {
		namespace += "/"
	}
	dockerfile := fmt.Sprintf(`
FROM scratch
LABEL %s=%s
LABEL maintainers="%v"
COPY / /
`, internal.ImageLabel, meta.Name, meta.Maintainers)
	df := filepath.Join(app.Path, "__Dockerfile-docker-app__")
	if err := ioutil.WriteFile(df, []byte(dockerfile), 0644); err != nil {
		return "", errors.Wrapf(err, "cannot create file %s", df)
	}
	defer os.Remove(df)
	di := filepath.Join(app.Path, ".dockerignore")
	if err := ioutil.WriteFile(di, []byte("__Dockerfile-docker-app__\n.dockerignore"), 0644); err != nil {
		return "", errors.Wrapf(err, "cannot create file %s", di)
	}
	defer os.Remove(di)
	imageName := namespace + internal.AppNameFromDir(app.Name) + internal.AppExtension + ":" + tag
	args := []string{"build", "-t", imageName, "-f", df, app.Path}
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
	defer f.Close()
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
			img, err := splitImageName(repotag)
			if err != nil {
				return err
			}
			appName := img.Name
			err = ioutil.WriteFile(filepath.Join(outputDir, internal.DirNameFromAppName(appName)), data, 0644)
			return errors.Wrap(err, "error writing output file")
		}
	}
	return fmt.Errorf("failed to find our layer in tarball")
}

// Push pushes an app to a registry
func Push(app *types.App, namespace, tag string) error {
	imageName, err := Save(app, namespace, tag)
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
	if err := pullImage(repotag); err != nil {
		return err
	}
	return Load(repotag, ".")
}

func pullImage(repotag string) error {
	cmd := exec.Command("docker", "pull", repotag)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "error pulling image %s", repotag)
	}
	return nil
}

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
