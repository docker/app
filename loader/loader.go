package loader

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/app/internal"
	"github.com/docker/app/specification"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli/compose/loader"
	"github.com/docker/cli/cli/compose/schema"
	"github.com/docker/docker/pkg/archive"
	"github.com/pkg/errors"
)

var (
	crlf       = []byte{'\r', '\n'}
	lf         = []byte{'\n'}
	delimiters = [][]byte{
		[]byte("\r\n---\r\n"),
		[]byte("\n---\r\n"),
		[]byte("\r\n---\n"),
		[]byte("\n---\n"),
	}
)

// useCRLF detects which line break should be used
func useCRLF(data []byte) bool {
	nbCrlf := bytes.Count(data, crlf)
	nbLf := bytes.Count(data, lf)
	switch {
	case nbCrlf == nbLf:
		// document contains only CRLF
		return true
	case nbCrlf == 0:
		// document does not contain any CRLF
		return false
	default:
		// document contains mixed line breaks, so use the OS default
		return bytes.Equal(defaultLineBreak, crlf)
	}
}

// splitSingleFile split a multidocument using all possible document delimiters
func splitSingleFile(data []byte) [][]byte {
	parts := [][]byte{data}
	for _, delimiter := range delimiters {
		var intermediate [][]byte
		for _, part := range parts {
			intermediate = append(intermediate, bytes.Split(part, delimiter)...)
		}
		parts = intermediate
	}
	return parts
}

// LoadFromSingleFile loads a docker app from a single-file format (as a reader)
func LoadFromSingleFile(path string, r io.Reader, ops ...func(*types.App) error) (*types.App, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrap(err, "error reading single-file")
	}

	parts := splitSingleFile(data)
	if len(parts) != 3 {
		return nil, errors.Errorf("malformed single-file application: expected 3 documents, got %d", len(parts))
	}

	var (
		metadata io.Reader
		compose  io.Reader
		params   io.Reader
	)
	for i := 0; i < 3; i++ {
		parsed, err := loader.ParseYAML(parts[i])
		if err != nil {
			return nil, err
		}
		if err := specification.Validate(parsed, internal.MetadataVersion); metadata == nil && err == nil {
			metadata = bytes.NewBuffer(parts[i])
		} else if err2 := schema.Validate(parsed, schema.Version(parsed)); compose == nil && err2 == nil {
			compose = bytes.NewBuffer(parts[i])
		} else if params == nil {
			params = bytes.NewBuffer(parts[i])
		} else {
			return nil, errors.New("malformed single-file application")
		}
	}

	appOps := append([]func(*types.App) error{
		types.WithComposes(compose),
		types.WithParameters(params),
		types.Metadata(metadata),
		types.WithCRLF(useCRLF(data)),
	}, ops...)
	return types.NewApp(path, appOps...)
}

// LoadFromDirectory loads a docker app from a directory
func LoadFromDirectory(path string, ops ...func(*types.App) error) (*types.App, error) {
	if _, err := os.Stat(filepath.Join(path, internal.ParametersFileName)); os.IsNotExist(err) {
		if _, err := os.Stat(filepath.Join(path, internal.DeprecatedSettingsFileName)); err == nil {
			return nil, errors.Errorf("\"settings.yml\" has been deprecated in favor of \"parameters.yml\"; please rename \"settings.yml\" to \"parameters.yml\"")
		}
	}
	return types.NewAppFromDefaultFiles(path, ops...)
}

// LoadFromTar loads a docker app from a tarball
func LoadFromTar(tar string, ops ...func(*types.App) error) (*types.App, error) {
	f, err := os.Open(tar)
	if err != nil {
		return nil, errors.Wrap(err, "cannot load app from tar")
	}
	defer f.Close()
	appOps := append(ops, types.WithPath(tar))
	return LoadFromTarReader(f, appOps...)
}

// LoadFromTarReader loads a docker app from a tarball reader
func LoadFromTarReader(r io.Reader, ops ...func(*types.App) error) (*types.App, error) {
	dir, err := ioutil.TempDir("", "load-from-tar")
	if err != nil {
		return nil, errors.Wrap(err, "cannot load app from tar")
	}
	if err := archive.Untar(r, dir, &archive.TarOptions{
		NoLchown: true,
	}); err != nil {
		originalErr := errors.Wrap(err, "cannot load app from tar")
		if err := os.RemoveAll(dir); err != nil {
			return nil, errors.Wrapf(originalErr, "cannot remove temporary folder : %s", err.Error())
		}
		return nil, originalErr
	}
	appOps := append([]func(*types.App) error{
		types.WithCleanup(func() {
			os.RemoveAll(dir)
		}),
	}, ops...)
	return LoadFromDirectory(dir, appOps...)
}
