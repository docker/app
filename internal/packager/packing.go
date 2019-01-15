package packager

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	"github.com/docker/docker/pkg/archive"
)

var dockerFile = `FROM docker/cnab-app-base:` + internal.Version + `
COPY . .`

const dockerIgnore = "Dockerfile"

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
func PackInvocationImageContext(app *types.App, target io.Writer) error {
	tarout := tar.NewWriter(target)
	defer tarout.Close()
	prefix := fmt.Sprintf("%s%s/", app.Metadata().Name, internal.AppExtension)
	if len(app.Composes()) != 1 {
		return errors.New("app should have one and only one compose file")
	}
	if len(app.ParametersRaw()) != 1 {
		return errors.New("app should have one and only parameters file")
	}
	if err := tarAddBytes(tarout, "Dockerfile", []byte(dockerFile)); err != nil {
		return err
	}
	if err := tarAddBytes(tarout, ".dockerignore", []byte(dockerIgnore)); err != nil {
		return err
	}
	if err := tarAddBytes(tarout, prefix+internal.MetadataFileName, app.MetadataRaw()); err != nil {
		return err
	}
	if err := tarAddBytes(tarout, prefix+internal.ComposeFileName, app.Composes()[0]); err != nil {
		return err
	}
	if err := tarAddBytes(tarout, prefix+internal.ParametersFileName, app.ParametersRaw()[0]); err != nil {
		return err
	}
	for _, attachment := range app.Attachments() {
		if err := tarAdd(tarout, prefix+attachment.Path(), filepath.Join(app.Path, attachment.Path())); err != nil {
			return err
		}
	}
	return nil
}

// Pack packs the app as a single file
func Pack(appname string, target io.Writer) error {
	tarout := tar.NewWriter(target)
	for _, f := range internal.FileNames {
		err := tarAdd(tarout, f, filepath.Join(appname, f))
		if err != nil {
			return err
		}
	}
	// check for images
	dir := "images"
	_, err := os.Stat(filepath.Join(appname, dir))
	if err == nil {
		if err := tarout.WriteHeader(&tar.Header{
			Typeflag: tar.TypeDir,
			Name:     dir,
			Mode:     0755,
		}); err != nil {
			return err
		}
		imageDir, err := os.Open(filepath.Join(appname, dir))
		if err != nil {
			return err
		}
		defer imageDir.Close()
		images, err := imageDir.Readdirnames(0)
		if err != nil {
			return err
		}
		for _, i := range images {
			err = tarAdd(tarout, filepath.Join(dir, i), filepath.Join(appname, dir, i))
			if err != nil {
				return err
			}
		}
	}
	return tarout.Close()
}

// Unpack extracts a packed app
func Unpack(appname, targetDir string) error {
	s, err := os.Stat(appname)
	if err != nil {
		// try appending our extension
		appname = internal.DirNameFromAppName(appname)
		s, err = os.Stat(appname)
	}
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("app already extracted")
	}
	out := filepath.Join(targetDir, internal.AppNameFromDir(appname)+internal.AppExtension)
	err = os.Mkdir(out, 0755)
	if err != nil {
		return err
	}
	f, err := os.Open(appname)
	if err != nil {
		return err
	}
	defer f.Close()
	return archive.Untar(f, out, &archive.TarOptions{
		NoLchown: true,
	})
}
