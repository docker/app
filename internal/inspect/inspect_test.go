package inspect

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/golden"
)

const (
	composeYAML = `version: "3.1"

services:
  web:
    image: nginx`
)

func TestInspect(t *testing.T) {
	dir := fs.NewDir(t, "inspect",
		fs.WithDir("no-maintainers",
			fs.WithFile(internal.ComposeFileName, composeYAML),
			fs.WithFile(internal.MetadataFileName, `
version: 0.1.0
name: foo`),
			fs.WithFile(internal.SettingsFileName, ``),
		),
		fs.WithDir("no-description",
			fs.WithFile(internal.ComposeFileName, composeYAML),
			fs.WithFile(internal.MetadataFileName, `
version: 0.1.0
name: foo
maintainers:
  - name: foo
    email: "foo@bar.com"`),
			fs.WithFile(internal.SettingsFileName, ""),
		),
		fs.WithDir("no-settings",
			fs.WithFile(internal.ComposeFileName, composeYAML),
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
			fs.WithFile(internal.ComposeFileName, composeYAML),
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
		app, err := types.NewAppFromDefaultFiles(dir.Join(appname))
		assert.NilError(t, err)
		err = Inspect(outBuffer, app)
		assert.NilError(t, err)
		golden.Assert(t, outBuffer.String(), fmt.Sprintf("inspect-%s.golden", appname), appname)
	}
}
