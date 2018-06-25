package packager

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/app/internal"
)

func tarAdd(tarout *tar.Writer, path, file string) error {
	payload, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	h := &tar.Header{
		Name:     path,
		Size:     int64(len(payload)),
		Mode:     0644,
		Typeflag: tar.TypeReg,
	}
	err = tarout.WriteHeader(h)
	if err != nil {
		return err
	}
	_, err = tarout.Write(payload)
	return err
}

func tarAddDir(tarout *tar.Writer, path string) error {
	h := &tar.Header{
		Name:     path,
		Size:     0,
		Typeflag: tar.TypeDir,
		Mode:     0755,
	}
	return tarout.WriteHeader(h)
}

// Pack packs the app as a single file
func Pack(appname string, target io.Writer) error {
	tarout := tar.NewWriter(target)
	files := []string{"metadata.yml", "docker-compose.yml", "settings.yml"}
	for _, f := range files {
		err := tarAdd(tarout, f, filepath.Join(appname, f))
		if err != nil {
			return err
		}
	}
	// check for images
	_, err := os.Stat(filepath.Join(appname, "images"))
	if err == nil {
		err = tarAddDir(tarout, "images")
		if err != nil {
			return err
		}
		imageDir, err := os.Open(filepath.Join(appname, "images"))
		if err != nil {
			return err
		}
		images, err := imageDir.Readdirnames(0)
		if err != nil {
			return err
		}
		for _, i := range images {
			err = tarAdd(tarout, filepath.Join("images", i), filepath.Join(appname, "images", i))
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
	return extract(appname, out)
}
