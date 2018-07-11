package render

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/app/internal"

	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/fs"
	"gotest.tools/golden"
)

func TestInspectErrorsOnFiles(t *testing.T) {
	dir := fs.NewDir(t, "inspect-errors",
		fs.WithDir("empty-app"),
		fs.WithDir("unparseable-metadata-app",
			fs.WithFile(internal.MetadataFileName, `something is wrong`),
		),
		fs.WithDir("no-settings-app",
			fs.WithFile(internal.MetadataFileName, `{}`),
		),
		fs.WithDir("unparseable-settings-app",
			fs.WithFile(internal.MetadataFileName, `{}`),
			fs.WithFile(internal.SettingsFileName, "foo"),
		),
	)
	defer dir.Remove()

	for appname, expectedError := range map[string]string{
		"inexistent-app":           "failed to read application metadata",
		"empty-app":                "failed to read application metadata",
		"unparseable-metadata-app": "failed to parse application metadat",
		"no-settings-app":          "failed to read application settings",
		"unparseable-settings-app": "failed to parse application settings",
	} {
		err := Inspect(ioutil.Discard, dir.Join(appname))
		assert.Check(t, is.ErrorContains(err, expectedError))
	}
}

func TestInspect(t *testing.T) {
	dir := fs.NewDir(t, "inspect",
		fs.WithDir("no-maintainers",
			fs.WithFile(internal.MetadataFileName, `
version: 0.1.0
name: foo`),
			fs.WithFile(internal.SettingsFileName, ``),
		),
		fs.WithDir("no-description",
			fs.WithFile(internal.MetadataFileName, `
version: 0.1.0
name: foo
maintainers:
  - name: foo
    email: "foo@bar.com"`),
			fs.WithFile(internal.SettingsFileName, ""),
		),
		fs.WithDir("no-settings",
			fs.WithFile(internal.MetadataFileName, `
version: 0.1.0
name: foo
maintainers:
  - name: foo
    email: "foo@bar.com"
description: "this is sparta !"`),
			fs.WithFile(internal.SettingsFileName, ""),
		),
		fs.WithDir("full",
			fs.WithFile(internal.MetadataFileName, `
version: 0.1.0
name: foo
maintainers:
  - name: foo
    email: "foo@bar.com"
description: "this is sparta !"`),
			fs.WithFile(internal.SettingsFileName, `
port: 8080
text: hello`),
		),
	)
	defer dir.Remove()

	for _, appname := range []string{
		"no-maintainers", "no-description", "no-settings", "full",
	} {
		outBuffer := new(bytes.Buffer)
		err := Inspect(outBuffer, dir.Join(appname))
		assert.NilError(t, err)
		golden.Assert(t, outBuffer.String(), fmt.Sprintf("inspect-%s.golden", appname))
	}
}
