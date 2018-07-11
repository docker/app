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

func tarToolchainVersion(tarout *tar.Writer) error {
	payload := []byte(internal.Version)
	if err := tarout.WriteHeader(&tar.Header{
		Name: internal.ToolchainVersionFile,
		Mode: 0644,
		Size: int64(len(payload)),
	}); err != nil {
		return err
	}
	_, err := tarout.Write(payload)
	return err
}

// Pack packs the app as a single file
func Pack(appname string, target io.Writer) error {
	tarout := tar.NewWriter(target)
	defer tarout.Close()
	return PackInto(appname, tarout)
}

// PackInto packs an image into an existing TarWriter
func PackInto(appname string, tarout *tar.Writer) error {
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
	// inject toolchain version
	return tarToolchainVersion(tarout)
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
