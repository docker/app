package e2e

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"strings"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"gotest.tools/assert"
	"gotest.tools/icmd"
)

func TestBuild(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		testDir := path.Join("testdata", "build")
		cmd.Command = dockerCli.Command("app", "build", path.Join(testDir, "single"), "--tag", "single:1.0.0")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cfg := getDockerConfigDir(t, cmd)

		f := path.Join(cfg, "app", "bundles", "docker.io", "library", "single", "_tags", "1.0.0", "bundle.json")
		data, err := ioutil.ReadFile(f)
		assert.NilError(t, err)
		var bndl bundle.Bundle
		err = json.Unmarshal(data, &bndl)
		assert.NilError(t, err)

		built := []string{bndl.InvocationImages[0].Digest, bndl.Images["web"].Digest, bndl.Images["worker"].Digest}
		for _, ref := range built {
			cmd.Command = dockerCli.Command("inspect", ref)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
		}
	})
}

func TestBuildWithoutTag(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		testDir := path.Join("testdata", "build")
		cmd.Command = dockerCli.Command("app", "build", path.Join(testDir, "single"))
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cfg := getDockerConfigDir(t, cmd)

		f := path.Join(cfg, "app", "bundles", "_ids")
		infos, err := ioutil.ReadDir(f)
		assert.NilError(t, err)
		assert.Equal(t, len(infos), 1)
		id := infos[0].Name()

		f = path.Join(cfg, "app", "bundles", "_ids", id, "bundle.json")
		data, err := ioutil.ReadFile(f)
		assert.NilError(t, err)
		var bndl bundle.Bundle
		err = json.Unmarshal(data, &bndl)
		assert.NilError(t, err)

		built := []string{bndl.InvocationImages[0].Digest, bndl.Images["web"].Digest, bndl.Images["worker"].Digest}
		for _, ref := range built {
			cmd.Command = dockerCli.Command("inspect", ref)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
		}
	})
}

func getDockerConfigDir(t *testing.T, cmd icmd.Cmd) string {
	var cfg string
	for _, s := range cmd.Env {
		if strings.HasPrefix(s, "DOCKER_CONFIG=") {
			cfg = s[14:]
		}
	}
	if cfg == "" {
		t.Fatalf("Failed to retrieve docker config folder")
	}
	return cfg
}
