package packager

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/pkg/archive"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/com"
	protobuf "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
)

// ExtractedApp represents a potentially extracted application package
type ExtractedApp struct {
	OriginalAppName string
	AppName         string
	Cleanup         func()
}

var (
	noop = func() {}
)

// findApp looks for an app in CWD or subdirs
func findApp(fs com.FrontServiceClient) (string, error) {
	flClient, err := fs.FileList(context.Background(), &protobuf.StringValue{Value: "."})
	if err != nil {
		return "", err
	}
	hit := ""
	multiHit := false
	for {
		s, err := flClient.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if strings.HasSuffix(s.Name, internal.AppExtension) {
			if hit != "" {
				multiHit = true
			} else {
				hit = s.Name
			}
		}
	}
	if multiHit {
		return "", fmt.Errorf("multiple applications found in current directory, specify the application name on the command line")
	}
	if hit == "" {
		return "", fmt.Errorf("no application found in current directory")
	}
	return hit, nil
}

// extractImage extracts a docker application in a docker image to a temporary directory
func extractImage(appname string) (ExtractedApp, error) {
	var imagename string
	if strings.Contains(appname, ":") {
		nametag := strings.Split(appname, ":")
		if len(nametag) == 3 || strings.Contains(nametag[1], "/") {
			nametag[1] = internal.DirNameFromAppName(nametag[1])
			appname = filepath.Base(nametag[1])
		} else {
			nametag[0] = internal.DirNameFromAppName(nametag[0])
			appname = filepath.Base(nametag[0])
		}
		imagename = strings.Join(nametag, ":")
	} else {
		imagename = internal.DirNameFromAppName(appname)
		appname = filepath.Base(imagename)
	}
	tempDir, err := ioutil.TempDir("", "dockerapp")
	if err != nil {
		return ExtractedApp{}, errors.Wrap(err, "failed to create temporary directory")
	}
	defer os.RemoveAll(tempDir)
	err = Load(imagename, tempDir)
	if err != nil {
		if !strings.Contains(imagename, "/") {
			return ExtractedApp{}, fmt.Errorf("could not locate application in either filesystem or docker image")
		}
		// Try to pull it
		cmd := exec.Command("docker", "pull", imagename)
		if err := cmd.Run(); err != nil {
			return ExtractedApp{}, fmt.Errorf("could not locate application in filesystem, docker image or registry")
		}
		if err := Load(imagename, tempDir); err != nil {
			return ExtractedApp{}, errors.Wrap(err, "failed to load pulled image")
		}
	}
	// this gave us a compressed app, run through extract again
	appname, cleanup, err := Extract(filepath.Join(tempDir, appname), nil)
	return ExtractedApp{"", appname, cleanup}, err
}

// Extract extracts the app content if it's an archive or single-file
func Extract(appname string, fs com.FrontServiceClient) (string, func(), error) {
	extracted, err := ExtractWithOrigin(appname, fs)
	return extracted.AppName, extracted.Cleanup, err
}

// ExtractWithOrigin extracts the app content if argument is an archive, or does nothing if a dir.
// It returns source file, effective app name, and cleanup function
// If appname is empty, it looks into cwd, and all subdirs for a single matching .dockerapp
// If nothing is found, it looks for an image and loads it
func ExtractWithOrigin(appname string, fs com.FrontServiceClient) (ExtractedApp, error) {
	if appname == "" {
		var err error
		if appname, err = findApp(fs); err != nil {
			return ExtractedApp{}, err
		}
	}
	if appname == "." {
		var err error
		if appname, err = os.Getwd(); err != nil {
			return ExtractedApp{}, errors.Wrap(err, "cannot resolve current working directory")
		}
	}
	originalAppname := appname
	appname = filepath.Clean(appname)
	// try appending our extension
	appname = internal.DirNameFromAppName(appname)

	// tar/untar
	tarDirClient, err := fs.TarDir(context.Background(), &protobuf.StringValue{Value: appname})
	if err != nil {
		return ExtractedApp{}, err
	}
	absAppPath, err := filepath.Abs(appname)
	if err != nil {
		return ExtractedApp{}, err
	}
	if err = os.MkdirAll(absAppPath, 0777); err != nil {
		return ExtractedApp{}, err
	}
	done := make(chan error)
	defer close(done)
	archiveReader, archiveWriter := io.Pipe()

	go func() {
		done <- archive.UntarUncompressed(archiveReader, absAppPath, &archive.TarOptions{
			NoLchown: true,
		})
	}()
	for {
		data, err := tarDirClient.Recv()
		if err == io.EOF {
			archiveWriter.Close()
			break
		}
		if err != nil {
			archiveWriter.CloseWithError(err)
			break
		}
		archiveWriter.Write(data.Value)
	}
	if err = <-done; err != nil {
		return ExtractedApp{}, err
	}
	s, err := os.Stat(absAppPath)
	if err != nil {
		// try verbatim
		s, err = os.Stat(originalAppname)
	}
	if err != nil {
		// look for a docker image
		return extractImage(originalAppname)
	}
	if s.IsDir() {
		// directory: already decompressed
		return ExtractedApp{appname, absAppPath, noop}, nil
	}
	// not a dir: single-file or a tarball package, extract that in a temp dir
	tempDir, err := ioutil.TempDir("", "dockerapp")
	if err != nil {
		return ExtractedApp{}, errors.Wrap(err, "failed to create temporary directory")
	}
	defer func() {
		if err != nil {
			os.RemoveAll(tempDir)
		}
	}()
	appDir := filepath.Join(tempDir, filepath.Base(appname))
	if err = os.Mkdir(appDir, 0755); err != nil {
		return ExtractedApp{}, errors.Wrap(err, "failed to create application in temporary directory")
	}
	if err = extract(absAppPath, appDir); err == nil {
		return ExtractedApp{appname, appDir, func() { os.RemoveAll(tempDir) }}, nil
	}
	if err = extractSingleFile(absAppPath, appDir); err != nil {
		return ExtractedApp{}, err
	}
	// not a tarball, single-file then
	return ExtractedApp{appname, appDir, func() { os.RemoveAll(tempDir) }}, nil
}

func extractSingleFile(appname, appDir string) error {
	// not a tarball, single-file then
	data, err := ioutil.ReadFile(appname)
	if err != nil {
		return errors.Wrap(err, "failed to read single-file application package")
	}
	parts := strings.Split(string(data), "\n---")
	if len(parts) != 3 {
		return fmt.Errorf("malformed single-file application: expected 3 documents")
	}
	for i, p := range parts {
		data := ""
		if i == 0 {
			data = p
		} else {
			d := strings.SplitN(p, "\n", 2)
			if len(d) > 1 {
				data = d[1]
			}
		}
		err = ioutil.WriteFile(filepath.Join(appDir, internal.FileNames[i]), []byte(data), 0644)
		if err != nil {
			return errors.Wrap(err, "failed to write application file")
		}
	}
	return nil
}

func extract(appname, outputDir string) error {
	f, err := os.Open(appname)
	if err != nil {
		return errors.Wrap(err, "failed to open application package")
	}
	defer f.Close()
	tarReader := tar.NewReader(f)
	outputDir = outputDir + "/"
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "error reading from tar header")
		}
		switch header.Typeflag {
		case tar.TypeDir: // = directory
			os.Mkdir(outputDir+header.Name, 0755)
		case tar.TypeReg: // = regular file
			data := make([]byte, header.Size)
			_, err := tarReader.Read(data)
			if err != nil && err != io.EOF {
				return errors.Wrap(err, "error reading from tar data")
			}
			err = ioutil.WriteFile(outputDir+header.Name, data, 0644)
			if err != nil {
				return errors.Wrap(err, "error writing output file")
			}
		}
	}
	return nil
}
