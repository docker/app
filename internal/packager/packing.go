package packager

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultCNABBaseImageName is the name of the default base invocation image.
	DefaultCNABBaseImageName = "docker/cnab-app-base"

	dockerIgnore = "Dockerfile"
)

func tarAdd(tarout *tar.Writer, path, file string) error {
	payload, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	return tarAddBytes(tarout, path, payload)
}

func tarAddBytes(tarout *tar.Writer, path string, payload []byte) error {
	h := &tar.Header{
		Name:     path,
		Size:     int64(len(payload)),
		Mode:     0644,
		Typeflag: tar.TypeReg,
	}
	err := tarout.WriteHeader(h)
	if err != nil {
		return err
	}
	_, err = tarout.Write(payload)
	return err
}

// PackInvocationImageContext creates a Docker build context for building a CNAB invocation image
func PackInvocationImageContext(cli command.Cli, app *types.App, target io.Writer) error {
	logrus.Debug("Packing invocation image context")
	tarout := tar.NewWriter(target)
	defer tarout.Close()
	prefix := fmt.Sprintf("%s%s/", app.Metadata().Name, internal.AppExtension)
	if len(app.Composes()) != 1 {
		return errors.New("app should have one and only one compose file")
	}
	if len(app.ParametersRaw()) != 1 {
		return errors.New("app should have one and only one parameters file")
	}
	if err := tarAddBytes(tarout, "Dockerfile", []byte(dockerFile(cli))); err != nil {
		return errors.Wrap(err, "failed to add Dockerfile to the invocation image build context")
	}
	if err := tarAddBytes(tarout, ".dockerignore", []byte(dockerIgnore)); err != nil {
		return errors.Wrap(err, "failed to add .dockerignore to the invocation image build context")
	}
	if err := tarAddBytes(tarout, prefix+internal.MetadataFileName, app.MetadataRaw()); err != nil {
		return errors.Wrapf(err, "failed to add %q to the invocation image build context", prefix+internal.MetadataFileName)
	}
	if err := tarAddBytes(tarout, prefix+internal.ComposeFileName, app.Composes()[0]); err != nil {
		return errors.Wrapf(err, "failed to add %q to the invocation image build context", prefix+internal.ComposeFileName)
	}
	if err := tarAddBytes(tarout, prefix+internal.ParametersFileName, app.ParametersRaw()[0]); err != nil {
		return errors.Wrapf(err, "failed to add %q to the invocation image build context", prefix+internal.ParametersFileName)
	}
	for _, attachment := range app.Attachments() {
		if err := tarAdd(tarout, prefix+attachment.Path(), filepath.Join(app.Path, attachment.Path())); err != nil {
			return errors.Wrapf(err, "failed to add attachment %q to the invocation image build context", prefix+attachment.Path())
		}
	}
	return nil
}

// BaseInvocationImage returns the name and tag of the CNAB base invocation image
func BaseInvocationImage(cli command.Cli) string {
	img := DefaultCNABBaseImageName + `:` + internal.Version
	if cfg := cli.ConfigFile(); cfg != nil {
		if v, ok := cfg.PluginConfig("app", "base-invocation-image"); ok {
			return v
		}
	}
	return img
}

func dockerFile(cli command.Cli) string {
	return fmt.Sprintf("FROM %s\nCOPY . .", BaseInvocationImage(cli))
}
