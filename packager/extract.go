package packager

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
)

// Extract extracts the app content if argument is an archive, or does nothing if a dir.
// It returns effective app name, and cleanup function
func Extract(appname string) (string, func(), error) {
	// try verbatim first
	s, err := os.Stat(appname)
	if err != nil {
		// try appending our extension
		appname = utils.DirNameFromAppName(appname)
		s, err = os.Stat(appname)
	}
	if err != nil {
		return "", func() {}, err
	}
	if s.IsDir() {
		// directory: already decompressed
		return appname, func() {}, nil
	}
	// not a dir: probably a tarball package, extract that in a temp dir
	f, err := os.Open(appname)
	if err != nil {
		return "", func() {}, err
	}
	tempDir, err := ioutil.TempDir("", "dockerapp")
	if err != nil {
		return "", func() {}, err
	}
	tarReader := tar.NewReader(f)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", func() {}, err
		}
		tempDir = tempDir + "/"
		switch header.Typeflag {
		case tar.TypeDir: // = directory
			os.Mkdir(tempDir+header.Name, 0755)
		case tar.TypeReg: // = regular file
			data := make([]byte, header.Size)
			_, err := tarReader.Read(data)
			if err != nil {
				return "", func() {}, err
			}
			ioutil.WriteFile(tempDir+header.Name, data, 0755)
		}
	}
	return tempDir, func() { os.RemoveAll(tempDir) }, nil
}
