package packager

import (
	"testing"

	"github.com/docker/app/internal"
	"github.com/docker/app/types/metadata"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestInitFromComposeFile(t *testing.T) {
	composeData := `
version: '3.0'
services:
  nginx:
    image: nginx:${NGINX_VERSION}
    command: nginx $NGINX_ARGS
`
	envData := "# some comment\nNGINX_VERSION=latest"

	inputDir := fs.NewDir(t, "app_input_",
		fs.WithFile(internal.ComposeFileName, composeData),
		fs.WithFile(".env", envData),
	)
	defer inputDir.Remove()

	appName := "my.dockerapp"
	dir := fs.NewDir(t, "app_",
		fs.WithDir(appName),
	)
	defer dir.Remove()

	err := initFromComposeFile(dir.Join(appName), inputDir.Join(internal.ComposeFileName))
	assert.NilError(t, err)

	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile(internal.ComposeFileName, composeData, fs.WithMode(0644)),
		fs.WithFile(internal.ParametersFileName, "NGINX_ARGS: FILL ME\nNGINX_VERSION: latest\n", fs.WithMode(0644)),
	)

	assert.Assert(t, fs.Equal(dir.Join(appName), manifest))
}

func TestInitFromInvalidComposeFile(t *testing.T) {
	err := initFromComposeFile("my.dockerapp", "doesnotexist")
	assert.ErrorContains(t, err, "failed to read")
}

func TestInitFromV2ComposeFile(t *testing.T) {
	composeData := `
version: '2.4'
services:
  nginx:
    image: nginx:${NGINX_VERSION}
    command: nginx $NGINX_ARGS
`
	inputDir := fs.NewDir(t, "app_input_",
		fs.WithFile(internal.ComposeFileName, composeData),
	)
	defer inputDir.Remove()

	appName := "my.dockerapp"
	dir := fs.NewDir(t, "app_",
		fs.WithDir(appName),
	)
	defer dir.Remove()

	err := initFromComposeFile(dir.Join(appName), inputDir.Join(internal.ComposeFileName))
	assert.ErrorContains(t, err, "unsupported Compose file version")
}

func TestInitFromV1ComposeFile(t *testing.T) {
	composeData := `
nginx:
  image: nginx
`
	inputDir := fs.NewDir(t, "app_input_",
		fs.WithFile(internal.ComposeFileName, composeData),
	)
	defer inputDir.Remove()

	appName := "my.dockerapp"
	dir := fs.NewDir(t, "app_",
		fs.WithDir(appName),
	)
	defer dir.Remove()

	err := initFromComposeFile(dir.Join(appName), inputDir.Join(internal.ComposeFileName))
	assert.ErrorContains(t, err, "unsupported Compose file version")
}

func TestWriteMetadataFile(t *testing.T) {
	appName := "writemetadata_test"
	tmpdir := fs.NewDir(t, appName)
	defer tmpdir.Remove()

	err := writeMetadataFile(appName, tmpdir.Path(), "", []string{"bearclaw:bearclaw"})
	assert.NilError(t, err)

	data := `# Version of the application
version: 0.1.0
# Name of the application
name: writemetadata_test
# A short description of the application
description: 
# List of application maintainers with name and email for each
maintainers:
  - name: bearclaw
    email: bearclaw
`

	manifest := fs.Expected(t,
		fs.WithFile(internal.MetadataFileName, data, fs.WithMode(0644)),
	)
	assert.Assert(t, fs.Equal(tmpdir.Path(), manifest))
}

func TestParseMaintainersData(t *testing.T) {
	input := []string{
		"sakuya:sakuya.izayoi@touhou.jp",
		"marisa.kirisame",
		"Reimu Hakurei",
		"Hong Meiling:kurenai.misuzu@touhou.jp",
		"    :    ",
		"perfect:cherry:blossom",
	}

	expectedOutput := []metadata.Maintainer{
		{Name: "sakuya", Email: "sakuya.izayoi@touhou.jp"},
		{Name: "marisa.kirisame", Email: ""},
		{Name: "Reimu Hakurei", Email: ""},
		{Name: "Hong Meiling", Email: "kurenai.misuzu@touhou.jp"},
		{Name: "    ", Email: "    "},
		{Name: "perfect", Email: "cherry:blossom"},
	}
	output := parseMaintainersData(input)

	assert.DeepEqual(t, output, expectedOutput)
}
