package packager

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/types"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// Save saves an app to docker and returns the image name.
func Save(appname, namespace, tag string) (string, error) {
	app, err := Extract(appname)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()
	metaFile := filepath.Join(app.AppName, internal.MetadataFileName)
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
	df := filepath.Join(app.AppName, "__Dockerfile-docker-app__")
	if err := ioutil.WriteFile(df, []byte(dockerfile), 0644); err != nil {
		return "", errors.Wrapf(err, "cannot create file %s", df)
	}
	defer os.Remove(df)
	di := filepath.Join(app.AppName, ".dockerignore")
	if err := ioutil.WriteFile(di, []byte("__Dockerfile-docker-app__\n.dockerignore"), 0644); err != nil {
		return "", errors.Wrapf(err, "cannot create file %s", di)
	}
	defer os.Remove(di)
	imageName := namespace + internal.AppNameFromDir(app.AppName) + internal.AppExtension + ":" + tag
	args := []string{"build", "-t", imageName, "-f", df, app.AppName}
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
	repoComps := strings.Split(repotag, ":")
	repo := repoComps[0]
	if len(repoComps) == 3 || (len(repoComps) == 2 && strings.Contains(repoComps[1], "/")) {
		repo = repoComps[1]
	}
	tempDir, err := ioutil.TempDir("", "dockerapp-load")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	if err := archive.Untar(f, tempDir, &archive.TarOptions{
		NoLchown: true,
	}); err != nil {
		return errors.Wrap(err, "failed to extract image")
	}
	return filepath.Walk(tempDir, func(path string, fi os.FileInfo, err error) error {
		if fi.Name() == "layer.tar" {
			return os.Rename(path, filepath.Join(outputDir, internal.DirNameFromAppName(filepath.Base(repo))))
		}
		return nil
	})
}

// Push pushes an app to a registry
func Push(appname, namespace, tag string) error {
	app, err := Extract(appname)
	if err != nil {
		return err
	}
	defer app.Cleanup()
	imageName, err := Save(app.AppName, namespace, tag)
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
