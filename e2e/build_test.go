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
		cmd.Command = dockerCli.Command("app", "build", path.Join(testDir, "single"), "single:1.0.0")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		var cfg string
		for _, s := range cmd.Env {
			if strings.HasPrefix(s, "DOCKER_CONFIG=") {
				cfg = s[14:]
			}
		}
		if cfg == "" {
			t.Fatalf("Failed to retrieve docker config folder")
		}

		f := path.Join(cfg, "app", "bundles", "docker.io", "library", "single", "_tags", "1.0.0.json")
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
