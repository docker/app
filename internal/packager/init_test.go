package packager

import (
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/app/internal/utils"
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

	err := writeMetadataFile(appName, tmpdir.Path(), "", []string{"bearclaw:bearclaw"})
	assert.NilError(t, err)

	data := `# Version of the application
version: 0.1.0
# Name of the application
name: writemetadata_test
# A short description of the application
description: 
# Repository prefix to use when pushing to a registry. This is typically your Hub username.
#repository_prefix: myHubUsername
# List of application maitainers with name and email for each
maintainers:
  - name: bearclaw
    email: bearclaw
# Specify false here if your application doesn't support Swarm or Kubernetes
targets:
  swarm: true
  kubernetes: true
`
	assert.NilError(t, err)

	manifest := fs.Expected(
		t,
		fs.WithFile("metadata.yml", data, fs.WithMode(0644)),
	)
	assert.Assert(t, fs.Equal(tmpdir.Path(), manifest))
}
