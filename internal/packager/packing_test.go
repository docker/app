package packager

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/docker/app/types"
	"github.com/docker/cli/cli/command"
	"gotest.tools/assert"
)

func TestPackInvocationImageContext(t *testing.T) {
	app, err := types.NewAppFromDefaultFiles("testdata/packages/packing.dockerapp")
	assert.NilError(t, err)
	buf := bytes.NewBuffer(nil)
	dockerCli, err := command.NewDockerCli()
	assert.NilError(t, err)
	assert.NilError(t, PackInvocationImageContext(dockerCli, app, buf))
	assert.NilError(t, hasExpectedFiles(buf, map[string]bool{
		"Dockerfile":                                              true,
		".dockerignore":                                           true,
		"packing.dockerapp/metadata.yml":                          true,
		"packing.dockerapp/docker-compose.yml":                    true,
		"packing.dockerapp/parameters.yml":                        true,
		"packing.dockerapp/config.cfg":                            true,
		"packing.dockerapp/nesteddir/config2.cfg":                 true,
		"packing.dockerapp/nesteddir/nested2/nested3/config3.cfg": true,
	}))
}

func hasExpectedFiles(r io.Reader, expectedFiles map[string]bool) error {
	tr := tar.NewReader(r)
	var errors []string
	originalExpectedFilesCount := len(expectedFiles)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		if hdr.Size == 0 {
			errors = append(errors, fmt.Sprintf("content of '%s' is empty", hdr.Name))
		}
		if _, ok := expectedFiles[hdr.Name]; !ok {
			errors = append(errors, fmt.Sprintf("couldn't find file '%s' in the tar archive", hdr.Name))
			continue
		}
		delete(expectedFiles, hdr.Name)
	}
	if len(expectedFiles) != 0 {
		errors = append(errors, fmt.Sprintf("number of expected files is in archive is '%d', but just '%d' were found",
			originalExpectedFilesCount, originalExpectedFilesCount-len(expectedFiles)))
		for k := range expectedFiles {
			errors = append(errors, fmt.Sprintf("expected file '%s' not found", k))
		}
	}
	if len(errors) != 0 {
		return fmt.Errorf("%s", strings.Join(errors, "\n"))
	}
	return nil
}
