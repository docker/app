package packager

import (
	"fmt"
	"os/user"
	"testing"

	"github.com/docker/app/internal"
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

	err := initFromComposeFile(nil, dir.Join(appName), inputDir.Join(internal.ComposeFileName))
	assert.NilError(t, err)

	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile(internal.ComposeFileName, composeData, fs.WithMode(0644)),
		fs.WithFile(internal.ParametersFileName, "NGINX_ARGS: FILL ME\nNGINX_VERSION: latest\n", fs.WithMode(0644)),
	)

	assert.Assert(t, fs.Equal(dir.Join(appName), manifest))
}

func TestInitFromComposeFileWithFlattenedParams(t *testing.T) {
	composeData := `
version: '3.0'
services:
  service1:
    ports:
      - ${ports.service1:-9001}
  service2:
    ports:
      - ${ports.service2-9002}
  service3:
    ports:
      - ${ports.service3:?'port is unset or empty in the environment'}
  service4:
    ports:
      - ${ports.service4?'port is unset or empty in the environment'}
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

	err := initFromComposeFile(nil, dir.Join(appName), inputDir.Join(internal.ComposeFileName))
	assert.NilError(t, err)

	const expectedParameters = `ports:
  service1: 9001
  service2: 9002
  service3: FILL ME
  service4: FILL ME
`
	const expectedUpdatedComposeData = `
version: '3.0'
services:
  service1:
    ports:
      - ${ports.service1}
  service2:
    ports:
      - ${ports.service2}
  service3:
    ports:
      - ${ports.service3}
  service4:
    ports:
      - ${ports.service4}
`
	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile(internal.ComposeFileName, expectedUpdatedComposeData, fs.WithMode(0644)),
		fs.WithFile(internal.ParametersFileName, expectedParameters, fs.WithMode(0644)),
	)
	assert.Assert(t, fs.Equal(dir.Join(appName), manifest))
}

func TestInitFromInvalidComposeFile(t *testing.T) {
	err := initFromComposeFile(nil, "my.dockerapp", "doesnotexist")
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

	err := initFromComposeFile(nil, dir.Join(appName), inputDir.Join(internal.ComposeFileName))
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

	err := initFromComposeFile(nil, dir.Join(appName), inputDir.Join(internal.ComposeFileName))
	assert.ErrorContains(t, err, "unsupported Compose file version")
}

func TestWriteMetadataFile(t *testing.T) {
	appName := "writemetadata_test"
	tmpdir := fs.NewDir(t, appName)
	defer tmpdir.Remove()

	err := writeMetadataFile(appName, tmpdir.Path())
	assert.NilError(t, err)

	userData, _ := user.Current()
	currentUser := ""
	if userData != nil {
		currentUser = userData.Username
	}

	data := fmt.Sprintf(`# Version of the application
version: 0.1.0
# Name of the application
name: writemetadata_test
# A short description of the application
description: 
# List of application maintainers with name and email for each
maintainers:
  - name: %s
    email: 
`, currentUser)

	manifest := fs.Expected(t,
		fs.WithFile(internal.MetadataFileName, data, fs.WithMode(0644)),
	)
	assert.Assert(t, fs.Equal(tmpdir.Path(), manifest))
}
