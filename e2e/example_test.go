package e2e

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/app/internal"
	_ "github.com/docker/cli/cli/compose/schema/data"
	"gotest.tools/assert"
	"gotest.tools/icmd"
)

const (
	metadataJSONSchemaVersion = "v0.2"
	metadataJSONSchema        = "specification/schemas/metadata_schema_" + metadataJSONSchemaVersion + ".json"
	composeJSONSchemaVersion  = "v3.6"
	composeJSONSchema         = "vendor/github.com/docker/cli/cli/compose/schema/data/config_schema_" + composeJSONSchemaVersion + ".json"
)

func TestExamplesAreValid(t *testing.T) {
	filepath.Walk("../examples", func(p string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(path.Base(p), internal.AppExtension) {
			return nil
		}
		validateRenderedComposeFile(t, p)
		if info.IsDir() {
			p = path.Join(p, internal.MetadataFileName)
		}
		validateMetadata(t, p)
		return filepath.SkipDir
	})
}

func validateMetadata(t *testing.T, p string) {
	data, err := ioutil.ReadFile(p)
	assert.NilError(t, err)
	validateYaml(t, data, metadataJSONSchema, p)
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
	validateYaml(t, buf.Bytes(), composeJSONSchema, p)
}

func validateYaml(t *testing.T, yaml []byte, schema string, file string) {
	schema = filepath.Join("..", schema)
	yamlschema := getYamlschemaBinary(t)
	cmd := icmd.Cmd{
		Command: []string{yamlschema, "-", schema},
		Stdin:   bytes.NewBuffer(yaml),
	}
	result := icmd.RunCmd(cmd)
	assert.NilError(t, result.Error, "failed to validate %s: %s", file, result.Stderr())
	assert.Equal(t, result.ExitCode, 0, "failed to validate %s: %s", file, result.Stderr())
}
