package packager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"

	"github.com/deislabs/duffle/pkg/loader"
)

var (
	// ErrNoArtifactsDirectory indicates a missing artifacts/ directory
	ErrNoArtifactsDirectory = errors.New("No artifacts/ directory found")
)

// Importer is responsible for importing a file
type Importer struct {
	Source      string
	Destination string
	Client      *client.Client
	Loader      loader.Loader
	Verbose     bool
}

// NewImporter creates a new secure *Importer
//
// source is the filesystem path to the archive.
// destination is the directory to unpack the contents.
// load is a loader.Loader preconfigured for loading secure or insecure bundles.
func NewImporter(source, destination string, load loader.Loader, verbose bool) (*Importer, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(context.Background())
	if err != nil {
		return nil, fmt.Errorf("cannot negotation Docker client version: %v", err)
	}

	return &Importer{
		Source:      source,
		Destination: destination,
		Client:      cli,
		Loader:      load,
		Verbose:     verbose,
	}, nil
}

// Import decompresses a bundle from Source (location of the compressed bundle) and properly places artifacts in the correct location(s)
func (im *Importer) Import() error {
	baseDir := strings.TrimSuffix(filepath.Base(im.Source), ".tgz")
	dest := filepath.Join(im.Destination, baseDir)
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	reader, err := os.Open(im.Source)
	if err != nil {
		return err
	}
	defer reader.Close()

	tarOptions := &archive.TarOptions{
		Compression:      archive.Gzip,
		IncludeFiles:     []string{"."},
		IncludeSourceDir: true,
		// Issue #416
		NoLchown: true,
	}
	if err := archive.Untar(reader, dest, tarOptions); err != nil {
		return fmt.Errorf("untar failed: %s", err)
	}

	// We try to load a bundle.cnab file first, and fall back to a bundle.json
	ext := "cnab"
	if _, err := os.Stat(filepath.Join(dest, "bundle.cnab")); os.IsNotExist(err) {
		ext = "json"
	}

	_, err = im.Loader.Load(filepath.Join(dest, "bundle."+ext))
	if err != nil {
		removeErr := os.RemoveAll(dest)
		if removeErr != nil {
			return fmt.Errorf("failed to load and validate bundle.%s on import %s and failed to remove invalid bundle from filesystem %s", ext, err, removeErr)
		}
		return fmt.Errorf("failed to load and validate bundle.%s: %s", ext, err)
	}

	artifactsDir := filepath.Join(dest, "artifacts")
	_, err = os.Stat(artifactsDir)
	if err == nil {
		filepath.Walk(artifactsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			out, err := im.Client.ImageLoad(context.Background(), file, false)
			if err != nil {
				return err
			}
			defer out.Body.Close()

			if im.Verbose {
				io.Copy(os.Stdout, out.Body)
			}

			return nil
		})
	}

	return nil
}
