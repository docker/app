package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/app/internal/relocated"

	"gotest.tools/assert"
	"gotest.tools/icmd"
)

func TestRelocationMapCreatedOnPull(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		cfg := getDockerConfigDir(t, cmd)

		path := filepath.Join("testdata", "local")
		ref := info.registryAddress + "/test/local:a-tag"
		bundlePath := filepath.Join(cfg, "app", "bundles", strings.Replace(info.registryAddress, ":", "_", 1), "test", "local", "_tags", "a-tag")

		// Given a pushed application
		build(t, cmd, dockerCli, ref, path)
		cmd.Command = dockerCli.Command("app", "push", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		// And given application files are remove
		assert.NilError(t, os.RemoveAll(bundlePath))
		_, err := os.Stat(filepath.Join(bundlePath, relocated.BundleFilename))
		assert.Assert(t, os.IsNotExist(err))

		// When application is pulled
		cmd.Command = dockerCli.Command("app", "pull", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// Then the relocation map should exist
		_, err = os.Stat(filepath.Join(bundlePath, relocated.RelocationMapFilename))
		assert.NilError(t, err)
	})
}
