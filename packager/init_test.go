package packager

import (
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/fs"

	"github.com/docker/lunchbox/utils"
	yaml "gopkg.in/yaml.v2"
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
	composeData := `services:
  nginx:
    image: nginx:${NGINX_VERSION}
    command: nginx $NGINX_ARGS
`
	envData := "# some comment\nNGINX_VERSION=latest"
	inputDir := randomName("app_input_")
	os.Mkdir(inputDir, 0755)
	ioutil.WriteFile(filepath.Join(inputDir, "docker-compose.yml"), []byte(composeData), 0644)
	ioutil.WriteFile(filepath.Join(inputDir, ".env"), []byte(envData), 0644)
	defer os.RemoveAll(inputDir)

	testAppName := randomName("app_")
	dirName := utils.DirNameFromAppName(testAppName)
	err := os.Mkdir(dirName, 0755)
	assert.NilError(t, err)
	defer os.RemoveAll(dirName)

	err = initFromComposeFile(testAppName, filepath.Join(inputDir, "docker-compose.yml"))
	assert.NilError(t, err)

	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile("docker-compose.yml", composeData, fs.WithMode(0644)),
		fs.WithFile("settings.yml", "NGINX_ARGS: FILL ME\nNGINX_VERSION: latest\n", fs.WithMode(0644)),
	)

	assert.Assert(t, fs.Equal(dirName, manifest))
}

func TestInitFromInvalidComposeFile(t *testing.T) {
	testAppName := randomName("app_")
	dirName := utils.DirNameFromAppName(testAppName)
	err := os.Mkdir(dirName, 0755)
	assert.NilError(t, err)
	defer os.RemoveAll(dirName)

	err = initFromComposeFile(testAppName, "doesnotexist")
	assert.ErrorContains(t, err, "failed to read")
}

func TestWriteMetadataFile(t *testing.T) {
	appName := "writemetadata_test"
	tmpdir := fs.NewDir(t, appName)
	defer tmpdir.Remove()

	err := writeMetadataFile(appName, tmpdir.Path(), "", nil)
	assert.NilError(t, err)

	data, err := yaml.Marshal(newMetadata(appName, "", nil))
	assert.NilError(t, err)

	manifest := fs.Expected(
		t,
		fs.WithFile("metadata.yml", "",
			fs.WithMode(0644),
			fs.WithBytes(data),
		),
	)
	assert.Assert(t, fs.Equal(tmpdir.Path(), manifest))
}
