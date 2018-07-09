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
	"github.com/docker/app/internal/types"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

func appName(appname string) string {
	return internal.AppNameFromDir(appname)
}

// Save saves an app to docker and returns the image name.
func Save(appname, namespace, tag string) (string, error) {
	appname, cleanup, err := Extract(appname)
	if err != nil {
		return "", err
	}
	defer cleanup()
	metaFile := filepath.Join(appname, internal.MetadataFileName)
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
	dockerfile := []byte(fmt.Sprintf(`
FROM scratch
LABEL %s=%s
LABEL maintainers="%v"
LABEL %s=%s
COPY / /
`, internal.ImageLabel, meta.Name, meta.Maintainers, internal.ToolchainVersionLabel, internal.Version))

	reader, writer := io.Pipe()
	defer reader.Close()

	go func() {
		defer writer.Close()
		tw := tar.NewWriter(writer)
		defer tw.Close()
		defer tw.Flush()
		if err := PackInto(appname, tw); err != nil {
			writer.CloseWithError(err)
			return
		}
		// add dockerfile
		if err := tw.WriteHeader(&tar.Header{
			Name: "__Dockerfile-docker-app__",
			Mode: 0644,
			Size: int64(len(dockerfile)),
		}); err != nil {
			writer.CloseWithError(err)
			return
		}
		if _, err := tw.Write(dockerfile); err != nil {
			writer.CloseWithError(err)
			return
		}
		// add dockerignore
		ignorePayload := []byte("__Dockerfile-docker-app__\n.dockerignore")
		if err := tw.WriteHeader(&tar.Header{
			Name: ".dockerignore",
			Mode: 0644,
			Size: int64(len(ignorePayload)),
		}); err != nil {
			writer.CloseWithError(err)
			return
		}
		if _, err := tw.Write(ignorePayload); err != nil {
			writer.CloseWithError(err)
			return
		}
	}()
	imageName := namespace + appName(appname) + internal.AppExtension + ":" + tag
	args := []string{"build", "-t", imageName, "-f", "__Dockerfile-docker-app__", "-"}
	cmd := exec.Command("docker", args...)
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = os.Stderr
	cmd.Stdin = reader
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
			err = ioutil.WriteFile(filepath.Join(outputDir, internal.DirNameFromAppName(filepath.Base(repo))), data, 0644)
			return errors.Wrap(err, "error writing output file")
		}
	}
	return fmt.Errorf("failed to find our layer in tarball")
}

// Push pushes an app to a registry
func Push(appname, namespace, tag string) error {
	appname, cleanup, err := Extract(appname)
	if err != nil {
		return err
	}
	defer cleanup()
	imageName, err := Save(appname, namespace, tag)
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
