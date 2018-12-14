package loader

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/deislabs/duffle/pkg/bundle"
)

// UnsignedLoader loads a bundle.json that is not signed.
type UnsignedLoader struct{}

// NewUnsignedLoader creates a new *UnsignedLoader
//
// An UnsignedLoader can load an unsigned bundle, which is represented as a plain JSON file.
func NewUnsignedLoader() *UnsignedLoader {
	return &UnsignedLoader{}
}

// Load loads the given unsigned bundle.
func (l *UnsignedLoader) Load(filename string) (*bundle.Bundle, error) {
	b := &bundle.Bundle{}
	data, err := loadData(filename)
	if err != nil {
		return b, err
	}
	return l.LoadData(data)
}

// LoadData loads a Bundle from the given data.
//
// This loads an unsigned JSON bundle file into a *Bundle.
func (l *UnsignedLoader) LoadData(data []byte) (*bundle.Bundle, error) {
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
