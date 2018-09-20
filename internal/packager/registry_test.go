package packager

import (
	"path"
	"testing"

	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"gotest.tools/fs"
)

const (
	validMeta = `name: test-app
version: 0.1.0`
	validCompose = `version: "3.0"
services:
  web:
    image: nginx`
	validSettings = `foo: bar`
)

func TestSplitImageName(t *testing.T) {
	input := []string{
		"official.dockerapp",
		"touhou/reimu.dockerapp",
		"tagged.dockerapp:1.2.25",
		"touhou/sakuya.dockerapp:4.23",
		"private.registry.co.uk/docker/anne.boleyn/annulment.dockerapp:15.28",
	}

	output := []imageComponents{
		{Name: "official.dockerapp", Repository: "docker.io/library/official.dockerapp"},
		{Name: "reimu.dockerapp", Repository: "docker.io/touhou/reimu.dockerapp"},
		{Name: "tagged.dockerapp", Repository: "docker.io/library/tagged.dockerapp", Tag: "1.2.25"},
		{Name: "sakuya.dockerapp", Repository: "docker.io/touhou/sakuya.dockerapp", Tag: "4.23"},
		{Name: "annulment.dockerapp", Repository: "private.registry.co.uk/docker/anne.boleyn/annulment.dockerapp", Tag: "15.28"},
	}

	for i, item := range input {
		out, err := splitImageName(item)
		assert.NilError(t, err, item)
		assert.DeepEqual(t, out, &output[i])
	}

	invalids := []string{
		"__.dockerapp",
		"colon:colon:colon.dockerapp:colon",
		"nametag.dockerapp:",
		"ends/with/slash/",
	}

	for _, item := range invalids {
		_, err := splitImageName(item)
		assert.ErrorContains(t, err, "failed to parse image name", item)
	}
}

func TestPushPayload(t *testing.T) {
	dir := fs.NewDir(t, "externalfile",
		fs.WithFile(internal.MetadataFileName, validMeta),
		fs.WithFile(internal.SettingsFileName, `foo: bar`),
		fs.WithFile(internal.ComposeFileName, validCompose),
		fs.WithFile("config.cfg", "something"),
		fs.WithDir("nesteddirectory",
			fs.WithFile("nestedconfig.cfg", "something"),
		),
	)
	defer dir.Remove()
	app, err := types.NewAppFromDefaultFiles(dir.Path())

	payload, err := createPayload(app)

	assert.NilError(t, err)
	assert.Assert(t, is.Len(payload, 5))
	assert.Assert(t, is.Equal(payload["config.cfg"], "something"))
	assert.Assert(t, is.Equal(payload[path.Join("nesteddirectory", "nestedconfig.cfg")], "something"))
}
