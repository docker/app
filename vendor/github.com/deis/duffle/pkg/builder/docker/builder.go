package docker

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/deis/duffle/pkg/builder"
	"github.com/deis/duffle/pkg/duffle/manifest"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/builder/dockerignore"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"

	"github.com/sirupsen/logrus"

	"golang.org/x/net/context"
)

const (
	// DockerignoreFilename is the filename for Docker's ignore file.
	DockerignoreFilename = ".dockerignore"
)

// Component contains all information to build a container image
type Component struct {
	name         string
	Image        string
	Dockerfile   string
	BuildContext io.ReadCloser

	dockerBuilder dockerBuilder
}

// Name is the component name
func (dc Component) Name() string {
	return dc.name
}

// Type represents the component type
func (dc Component) Type() string {
	return "docker"
}

// URI returns the image in the format <registry>/<image>
func (dc Component) URI() string {
	return dc.Image
}

// Digest returns the name of a Docker component, which will give the image name
//
// TODO - return the actual digest
func (dc Component) Digest() string {
	return strings.Split(dc.Image, ":")[1]
}

// NewComponent returns a new Docker component based on the manifest
func NewComponent(c *manifest.Component, cli *command.DockerCli) *Component {
	return &Component{
		name: c.Name,
		// TODO - handle different Dockerfile names
		Dockerfile:    "Dockerfile",
		dockerBuilder: dockerBuilder{DockerClient: cli},
	}
}

// Builder contains information about the Docker build environment
type dockerBuilder struct {
	DockerClient command.Cli
}

// PrepareBuild archives the component directory and loads it as Docker context
func (dc *Component) PrepareBuild(ctx *builder.Context) error {
	if err := archiveSrc(filepath.Join(ctx.AppDir, dc.name), dc); err != nil {
		return err
	}

	defer dc.BuildContext.Close()

	// write each build context to a buffer so we can also write to the sha256 hash.
	buf := new(bytes.Buffer)
	h := sha256.New()
	w := io.MultiWriter(buf, h)
	if _, err := io.Copy(w, dc.BuildContext); err != nil {
		return err
	}

	// truncate checksum to the first 40 characters (20 bytes) this is the
	// equivalent of `shasum build.tar.gz | awk '{print $1}'`.
	ctxtID := h.Sum(nil)
	imgtag := fmt.Sprintf("%.20x", ctxtID)
	imageRepository := path.Join(ctx.Manifest.Components[dc.Name()].Configuration["registry"], fmt.Sprintf("%s-%s", ctx.Manifest.Name, dc.Name()))
	dc.Image = fmt.Sprintf("%s:%s", imageRepository, imgtag)

	dc.BuildContext = ioutil.NopCloser(buf)

	return nil
}

// Build builds the docker images.
func (dc Component) Build(ctx context.Context, app *builder.AppContext) error {
	defer dc.BuildContext.Close()
	buildOpts := types.ImageBuildOptions{
		Tags:       []string{dc.Image},
		Dockerfile: dc.Dockerfile,
	}

	resp, err := dc.dockerBuilder.DockerClient.Client().ImageBuild(ctx, dc.BuildContext, buildOpts)
	if err != nil {
		return fmt.Errorf("error building component %v with builder %v: %v", dc.Name(), dc.Type(), err)
	}

	defer resp.Body.Close()
	outFd, isTerm := term.GetFdInfo(dc.BuildContext)
	if err := jsonmessage.DisplayJSONMessagesStream(resp.Body, app.Log, outFd, isTerm, nil); err != nil {
		return fmt.Errorf("error streaming messages for component %v with builder %v: %v", dc.Name(), dc.Type(), err)
	}

	if _, _, err = dc.dockerBuilder.DockerClient.Client().ImageInspectWithRaw(ctx, dc.Image); err != nil {
		if dockerclient.IsErrNotFound(err) {
			return fmt.Errorf("could not locate image for %s: %v", dc.Name(), err)
		}
		return fmt.Errorf("imageInspectWithRaw error for component %v: %v", dc.Name(), err)
	}

	return nil
}

func archiveSrc(contextPath string, component *Component) error {
	contextDir, relDockerfile, err := build.GetContextFromLocalDir(contextPath, "")
	if err != nil {
		return fmt.Errorf("unable to prepare docker context: %s", err)
	}

	// canonicalize dockerfile name to a platform-independent one
	relDockerfile = archive.CanonicalTarNameForPath(relDockerfile)

	f, err := os.Open(filepath.Join(contextDir, DockerignoreFilename))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer f.Close()

	var excludes []string
	if err == nil {
		excludes, err = dockerignore.ReadAll(f)
		if err != nil {
			return err
		}
	}

	if err := build.ValidateContextDirectory(contextDir, excludes); err != nil {
		return fmt.Errorf("error checking docker context: '%s'", err)
	}

	// If .dockerignore mentions .dockerignore or the Dockerfile
	// then make sure we send both files over to the daemon
	// because Dockerfile is, obviously, needed no matter what, and
	// .dockerignore is needed to know if either one needs to be
	// removed. The daemon will remove them for us, if needed, after it
	// parses the Dockerfile. Ignore errors here, as they will have been
	// caught by validateContextDirectory above.
	var includes = []string{"."}
	keepThem1, _ := fileutils.Matches(DockerignoreFilename, excludes)
	keepThem2, _ := fileutils.Matches(relDockerfile, excludes)
	if keepThem1 || keepThem2 {
		includes = append(includes, DockerignoreFilename, relDockerfile)
	}

	logrus.Debugf("INCLUDES: %v", includes)
	logrus.Debugf("EXCLUDES: %v", excludes)
	dockerArchive, err := archive.TarWithOptions(contextDir, &archive.TarOptions{
		ExcludePatterns: excludes,
		IncludeFiles:    includes,
	})
	if err != nil {
		return err
	}

	component.name = filepath.Base(contextDir)
	component.BuildContext = dockerArchive
	component.Dockerfile = relDockerfile

	return nil
}
