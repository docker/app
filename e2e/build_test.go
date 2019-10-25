package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"gotest.tools/fs"

	"github.com/docker/app/internal/store"

	"github.com/deislabs/cnab-go/bundle"
	"gotest.tools/assert"
	"gotest.tools/icmd"
)

func TestBuild(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		tmp := fs.NewDir(t, "TestBuild")

		testDir := path.Join("testdata", "build")
		iidfile := tmp.Join("iidfile")
		cmd.Command = dockerCli.Command("app", "build", "--tag", "single:1.0.0", "--iidfile", iidfile, "-f", path.Join(testDir, "single.dockerapp"), testDir)
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

		for _, img := range bndl.Images {
			// Check all image not being built locally get a fixed reference
			assert.Assert(t, img.Image == "" || strings.Contains(img.Image, "@sha256:"))
		}

		_, err = os.Stat(iidfile)
		assert.NilError(t, err)
		bytes, err := ioutil.ReadFile(iidfile)
		assert.NilError(t, err)
		iid := string(bytes)
		actualID, err := store.FromBundle(&bndl)
		assert.NilError(t, err)
		assert.Equal(t, iid, fmt.Sprintf("sha256:%s", actualID.String()))
	})
}

func TestQuietBuild(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		tmp := fs.NewDir(t, "TestBuild")

		testDir := path.Join("testdata", "build")
		iidfile := tmp.Join("iidfile")
		cmd.Command = dockerCli.Command("app", "build", "--quiet", "--iidfile", iidfile, "-f", path.Join(testDir, "single.dockerapp"), testDir)
		out := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
		out = strings.Trim(out, "\r\n")

		_, err := os.Stat(iidfile)
		assert.NilError(t, err)
		bytes, err := ioutil.ReadFile(iidfile)
		assert.NilError(t, err)
		iid := string(bytes)
		assert.NilError(t, err)
		assert.Equal(t, iid, out)
	})
}

func TestBuildWithoutTag(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		testDir := path.Join("testdata", "build")
		cmd.Command = dockerCli.Command("app", "build", "-f", path.Join(testDir, "single.dockerapp"), testDir)
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

func TestBuildWithArgs(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		testDir := path.Join("testdata", "build")
		cmd.Command = dockerCli.Command("app", "build", "-f", path.Join(testDir, "single.dockerapp"), testDir, "--build-arg", "REPLACE_BY_BUILD_ARG=replaced")
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

		cmd.Command = dockerCli.Command("inspect", bndl.Images["worker"].Digest)
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      `"com.docker.labelled.arg": "replaced"`,
		})
	})
}

func TestBuildWithArgsDefinedTwice(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		testDir := path.Join("testdata", "build")
		cmd.Command = dockerCli.Command("app", "build", "-f", path.Join(testDir, "single.dockerapp"), testDir,
			"--build-arg", "REPLACE_BY_BUILD_ARG=replaced", "--build-arg", "REPLACE_BY_BUILD_ARG=replaced_twice")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `'--build-arg REPLACE_BY_BUILD_ARG' is defined twice`,
		})
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
