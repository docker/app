package e2e

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/assert"

	"github.com/docker/app/internal"

	"gotest.tools/icmd"

	_ "github.com/docker/cli/cli/compose/schema/data"
)

const (
	metadataJsonSchemaVersion = "v0.2"
	metadataJsonSchema        = "specification/schemas/metadata_schema_" + metadataJsonSchemaVersion + ".json"
	composeJsonSchemaVersion  = "v3.6"
	composeJsonSchema         = "vendor/github.com/docker/cli/cli/compose/schema/data/config_schema_" + composeJsonSchemaVersion + ".json"
)

func TestExamplesAreValid(t *testing.T) {
	filepath.Walk("../examples", func(p string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(path.Base(p), internal.AppExtension) {
			return nil
		}
		if info.IsDir() {
			validateMetadata(t, path.Join(p, internal.MetadataFileName))
			validateRenderedComposeFile(t, p)
		} else {
			validateMetadata(t, p)
			validateRenderedComposeFile(t, p)
		}
		return filepath.SkipDir
	})
}

func validateMetadata(t *testing.T, p string) {
	data, err := ioutil.ReadFile(p)
	assert.NilError(t, err)
	validateYaml(t, data, "../"+metadataJsonSchema, p)
}

func validateRenderedComposeFile(t *testing.T, p string) {
	dockerApp, _ := getDockerAppBinary(t)
	buf := &bytes.Buffer{}
	cmd := icmd.Cmd{
		Command: []string{dockerApp, "render"},
		Dir:     filepath.Dir(p),
		Stdout:  buf,
	}
	result := icmd.RunCmd(cmd)
	result.Assert(t, icmd.Success)
	validateYaml(t, buf.Bytes(), "../"+composeJsonSchema, p)
}

func validateYaml(t *testing.T, yaml []byte, schema string, file string) {
	yamlschema := getYamlschemaBinary(t)
	cmd := icmd.Cmd{
		Command: []string{yamlschema, "-", schema},
		Stdin:   bytes.NewBuffer(yaml),
	}
	result := icmd.RunCmd(cmd)
	assert.NilError(t, result.Error, "failed to validate %s: %s", file, result.Stderr())
	assert.Equal(t, result.ExitCode, 0, "failed to validate %s: %s", file, result.Stderr())
}
