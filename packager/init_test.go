package packager

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/fs"

	"github.com/docker/lunchbox/types"
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

type DummyConfigMerger struct{}

func NewDummyConfigMerger() ComposeConfigMerger {
	return &DummyConfigMerger{}
}

var dummyComposeData = `
version: '3.6'
services:
  foo:
    image: bar
    command: baz
`

func (m *DummyConfigMerger) MergeComposeConfig(composeFiles []string) ([]byte, error) {
	if composeFiles[0] == "doesnotexist" {
		return []byte{}, fmt.Errorf("no file named %q", composeFiles[0])
	}
	return []byte(dummyComposeData), nil
}

func TestComposeFileFromScratch(t *testing.T) {
	services := []string{
		"redis", "mysql", "python",
	}

	result, err := composeFileFromScratch(services)
	assert.NilError(t, err)

	expected := types.NewInitialComposeFile()
	expected.Services = &map[string]types.InitialService{
		"redis":  {Image: "redis"},
		"mysql":  {Image: "mysql"},
		"python": {Image: "python"},
	}
	expectedBytes, err := yaml.Marshal(expected)
	assert.NilError(t, err)
	assert.DeepEqual(t, result, expectedBytes)
}

func TestInitFromComposeFiles(t *testing.T) {
	testAppName := randomName("app_")
	merger := NewDummyConfigMerger()
	dirName := utils.DirNameFromAppName(testAppName)
	err := os.Mkdir(dirName, 0755)
	assert.NilError(t, err)
	defer os.RemoveAll(dirName)

	err = initFromComposeFiles(testAppName, []string{"docker-compose.yml"}, merger)
	assert.NilError(t, err)

	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithFile("services.yml", dummyComposeData, fs.WithMode(0644)),
		fs.WithFile("settings.yml", "\n", fs.WithMode(0644)),
	)

	assert.Assert(t, fs.Equal(dirName, manifest))
}

func TestInitFromInvalidComposeFile(t *testing.T) {
	testAppName := randomName("app_")
	merger := NewDummyConfigMerger()
	dirName := utils.DirNameFromAppName(testAppName)
	err := os.Mkdir(dirName, 0755)
	assert.NilError(t, err)
	defer os.RemoveAll(dirName)

	err = initFromComposeFiles(testAppName, []string{"doesnotexist"}, merger)
	assert.ErrorContains(t, err, "no file named \"doesnotexist\"")
}

func TestWriteMetadataFile(t *testing.T) {
	appName := "writemetadata_test"
	tmpdir := fs.NewDir(t, appName)
	defer tmpdir.Remove()

	err := writeMetadataFile(appName, tmpdir.Path())
	assert.NilError(t, err)

	data, err := yaml.Marshal(newMetadata(appName))
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
