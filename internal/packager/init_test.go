package packager

import (
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/app/internal"
	"github.com/docker/app/types/metadata"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

func randomName(prefix string) string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return prefix + hex.EncodeToString(b)
}

func TestInitFromComposeFile(t *testing.T) {
	composeData := `
version: '3.0'
services:
  nginx:
    image: nginx:${NGINX_VERSION}
    command: nginx $NGINX_ARGS
`
	envData := "# some comment\nNGINX_VERSION=latest"
	inputDir := randomName("app_input_")
	os.Mkdir(inputDir, 0755)
	ioutil.WriteFile(filepath.Join(inputDir, internal.ComposeFileName), []byte(composeData), 0644)
	ioutil.WriteFile(filepath.Join(inputDir, ".env"), []byte(envData), 0644)
	defer os.RemoveAll(inputDir)

	testAppName := randomName("app_")
	dirName := internal.DirNameFromAppName(testAppName)
	err := os.Mkdir(dirName, 0755)
	assert.NilError(t, err)
	defer os.RemoveAll(dirName)

	err = initFromComposeFile(testAppName, filepath.Join(inputDir, internal.ComposeFileName))
	assert.NilError(t, err)

	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile(internal.ComposeFileName, composeData, fs.WithMode(0644)),
		fs.WithFile(internal.SettingsFileName, "NGINX_ARGS: FILL ME\nNGINX_VERSION: latest\n", fs.WithMode(0644)),
	)

	assert.Assert(t, fs.Equal(dirName, manifest))
}

func TestInitFromInvalidComposeFile(t *testing.T) {
	testAppName := randomName("app_")
	dirName := internal.DirNameFromAppName(testAppName)
	err := os.Mkdir(dirName, 0755)
	assert.NilError(t, err)
	defer os.RemoveAll(dirName)

	err = initFromComposeFile(testAppName, "doesnotexist")
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
	inputDir := randomName("app_input_")
	os.Mkdir(inputDir, 0755)
	ioutil.WriteFile(filepath.Join(inputDir, "docker-compose.yml"), []byte(composeData), 0644)
	defer os.RemoveAll(inputDir)

	testAppName := randomName("app_")
	dirName := internal.DirNameFromAppName(testAppName)
	err := os.Mkdir(dirName, 0755)
	assert.NilError(t, err)
	defer os.RemoveAll(dirName)

	err = initFromComposeFile(testAppName, filepath.Join(inputDir, "docker-compose.yml"))
	assert.ErrorContains(t, err, "unsupported Compose file version")
}

func TestInitFromV1ComposeFile(t *testing.T) {
	composeData := `
nginx:
  image: nginx
`
	inputDir := randomName("app_input_")
	os.Mkdir(inputDir, 0755)
	ioutil.WriteFile(filepath.Join(inputDir, "docker-compose.yml"), []byte(composeData), 0644)
	defer os.RemoveAll(inputDir)

	testAppName := randomName("app_")
	dirName := internal.DirNameFromAppName(testAppName)
	err := os.Mkdir(dirName, 0755)
	assert.NilError(t, err)
	defer os.RemoveAll(dirName)

	err = initFromComposeFile(testAppName, filepath.Join(inputDir, "docker-compose.yml"))
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
# Namespace to use when pushing to a registry. This is typically your Hub username.
#namespace: myHubUsername
# List of application maintainers with name and email for each
maintainers:
  - name: bearclaw
    email: bearclaw
`
	assert.NilError(t, err)

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
