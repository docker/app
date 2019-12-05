package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/app/internal/image"
	"gotest.tools/assert/cmp"

	"gotest.tools/assert"
	"gotest.tools/icmd"
)

func TestRelocationMapRun(t *testing.T) {
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
		_, err := os.Stat(filepath.Join(bundlePath, image.BundleFilename))
		assert.Assert(t, os.IsNotExist(err))
		// And given local images are removed
		cmd.Command = dockerCli.Command("rmi", "web", "local:1.1.0-beta1-invoc", "worker")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		// And given application is pulled from the registry
		cmd.Command = dockerCli.Command("app", "pull", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		t.Run("with-relocation-map", func(t *testing.T) {
			name := "test-relocation-map-run-with-relocation-map"
			// When application is run
			cmd.Command = dockerCli.Command("app", "run", "--name", name, ref)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			// Then the application is running
			cmd.Command = dockerCli.Command("app", "ls")
			assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), name))

			cmd.Command = dockerCli.Command("app", "rm", name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
		})

		t.Run("without-relocation-map", func(t *testing.T) {
			name := "test-relocation-map-run-without-relocation-map"
			// And given the relocation map is removed after the pull
			assert.NilError(t, filepath.Walk(filepath.Join(cfg, "app", "bundles", "contents"), func(path string, info os.FileInfo, err error) error {
				if info.Name() == image.RelocationMapFilename {
					os.Remove(path)
				}
				return nil
			}))

			// Then the application cannot be run
			cmd.Command = dockerCli.Command("app", "run", "--name", name, ref)
			icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 1})
		})
	})
}

func TestPushPulledApplication(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		cfg := getDockerConfigDir(t, cmd)

		path := filepath.Join("testdata", "local")
		ref := info.registryAddress + "/test/local:a-tag"

		// Given an application pushed on a registry
		build(t, cmd, dockerCli, ref, path)
		cmd.Command = dockerCli.Command("app", "push", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		// And given local images are removed
		assert.NilError(t, os.RemoveAll(filepath.Join(cfg, "app", "bundles")))

		// And given application is pulled from the registry
		cmd.Command = dockerCli.Command("app", "pull", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// Then the application can still be pushed
		cmd.Command = dockerCli.Command("app", "push", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
	})
}

func TestRelocationMapOnInspect(t *testing.T) {
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
		_, err := os.Stat(filepath.Join(bundlePath, image.BundleFilename))
		assert.Assert(t, os.IsNotExist(err))
		// And given local images are removed
		cmd.Command = dockerCli.Command("rmi", "web", "local:1.1.0-beta1-invoc", "worker")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		// And given application is pulled from the registry
		cmd.Command = dockerCli.Command("app", "pull", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// When inspect the image
		cmd.Command = dockerCli.Command("app", "image", "inspect", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
	})
}
