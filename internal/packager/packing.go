package packager

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/app/internal"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
)

// Pack packs the app as a single file
func Pack(appname string, target io.Writer) error {
	files := append([]string{}, internal.FileNames...)
	// Include image if present
	if _, err := os.Stat(filepath.Join(appname, "images")); err == nil {
		files = append(files, "images")
	}
	r, err := archive.TarWithOptions(appname, &archive.TarOptions{
		IncludeFiles: files,
		Compression:  archive.Uncompressed,
	})
	if err != nil {
		return errors.Wrap(err, "cannot create an archive")
	}
	_, err = io.Copy(target, r)
	return err
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
