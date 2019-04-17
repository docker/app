package loader

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/deislabs/cnab-go/bundle"
)

// Loader provides an interface for loading a bundle
type BundleLoader interface {
	// Load a bundle from a local file
	Load(source string) (*bundle.Bundle, error)
	// Load a bundle from raw data
	LoadData(data []byte) (*bundle.Bundle, error)
}

// Loader loads a bundle manifest (bundle.json)
type Loader struct{}

// New creates a loader for bundle files.
//TODO: remove if unnecessary
func New() BundleLoader {
	return &Loader{}
}

func NewLoader() *Loader {
	return &Loader{}
}

// Load loads the given bundle.
func (l *Loader) Load(filename string) (*bundle.Bundle, error) {
	b := &bundle.Bundle{}
	data, err := loadData(filename)
	if err != nil {
		return b, err
	}
	return l.LoadData(data)
}

// LoadData loads a Bundle from the given data.
//
// This loads a JSON bundle file into a *bundle.Bundle.
func (l *Loader) LoadData(data []byte) (*bundle.Bundle, error) {
	return bundle.Unmarshal(data)
}

// loadData is a utility method that loads a file either off of the FS (if it exists) or via a remote HTTP GET.
//
// If bundleFile exists on disk, this will return that file. Otherwise, it will attempt to parse the
// file name as a URL and request it as an HTTP GET request.
func loadData(bundleFile string) ([]byte, error) {
	if isLocalReference(bundleFile) {
		return ioutil.ReadFile(bundleFile)
	}

	if u, err := url.ParseRequestURI(bundleFile); err != nil {
		// The error emitted by ParseRequestURI is icky.
		return []byte{}, fmt.Errorf("bundle %q not found", bundleFile)
	} else if u.Scheme == "file" {
		// What do we do if someone passes a `file:///` URL in? Is `file` inferred
		// if no protocol is specified?
		return []byte{}, fmt.Errorf("bundle %q not found", bundleFile)
	}

	response, err := http.Get(bundleFile)
	if err != nil {
		return []byte{}, fmt.Errorf("cannot download bundle file: %v", err)
	}
	defer response.Body.Close()

	return ioutil.ReadAll(response.Body)
}

func isLocalReference(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}
