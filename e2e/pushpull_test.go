package e2e

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
	"gotest.tools/icmd"
)

func TestPushUnknown(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	t.Run("push unknown reference", func(t *testing.T) {
		cmd.Command = dockerCli.Command("app", "push", "unknown")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `could not push "unknown": unknown: reference not found`,
		})
	})

	t.Run("push invalid reference", func(t *testing.T) {
		cmd.Command = dockerCli.Command("app", "push", "@")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `could not push "@": could not parse "@" as a valid reference: invalid reference format`,
		})
	})
}

func TestPushInsecureRegistry(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		path := filepath.Join("testdata", "local")
		ref := info.registryAddress + "/test/push-insecure"

		// create a command outside of the dind context so without the insecure registry configured
		cmdNoInsecureRegistry, cleanupNoInsecureRegistryCommand := dockerCli.createTestCmd()
		defer cleanupNoInsecureRegistryCommand()
		build(t, cmdNoInsecureRegistry, dockerCli, ref, path)
		cmdNoInsecureRegistry.Command = dockerCli.Command("app", "push", ref)
		icmd.RunCmd(cmdNoInsecureRegistry).Assert(t, icmd.Expected{ExitCode: 1})

		// run the push with the command inside dind context configured to allow access to the insecure registr
		cmd := info.configuredCmd
		build(t, cmd, dockerCli, ref, path)
		cmd.Command = dockerCli.Command("app", "push", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
	})
}

func TestPushInstall(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		ref := info.registryAddress + "/test/push-pull"
		build(t, cmd, dockerCli, ref, filepath.Join("testdata", "push-pull"))

		cmd.Command = dockerCli.Command("app", "push", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "run", ref, "--name", t.Name())
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		cmd.Command = dockerCli.Command("service", "ls")
		assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), t.Name()))
	})
}

func TestPushPullInstall(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		ref := info.registryAddress + "/test/push-pull"
		tag := ":v.0.0.1"
		build(t, cmd, dockerCli, ref+tag, filepath.Join("testdata", "push-pull"))

		cmd.Command = dockerCli.Command("app", "push", ref+tag)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		cmd.Command = dockerCli.Command("app", "pull", ref+tag)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// stop the registry
		info.stopRegistry()

		// install from local store
		cmd.Command = dockerCli.Command("app", "run", ref+tag, "--name", t.Name())
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// listing the installed application shows the pulled application reference
		cmd.Command = dockerCli.Command("app", "ls")
		checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
			[]string{
				fmt.Sprintf(`%s\s+push-pull \(1.1.0-beta1\)\s+\d/1\s+install\s+success\s+.+second[s]?\sago\s+.+second[s]?\sago\s+%s`, t.Name(), ref+tag),
			})

		// install should fail (registry is stopped)
		cmd.Command = dockerCli.Command("app", "run", "unknown")
		//nolint: lll
		expected := `Unable to find App image "unknown:latest" locally
Unable to find App "unknown": failed to resolve bundle manifest "docker.io/library/unknown:latest": pull access denied, repository does not exist or may require authorization: server message: insufficient_scope: authorization failed`
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      expected,
			Out:      "Pulling from registry...",
		})
	})
}

func TestPushPullServiceImages(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		ref := info.registryAddress + "/test/push-pull"
		tag := ":v.0.0.1"
		build(t, cmd, dockerCli, ref+tag, filepath.Join("testdata", "push-pull"))

		cmd.Command = dockerCli.Command("app", "push", ref+tag)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// Make sure this image does not exist so that we can verify the pull works.
		cmd.Command = dockerCli.Command("image", "rm", "busybox:1.30.1")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "pull", "--service-images", ref+tag)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("image", "inspect", "busybox@sha256:4b6ad3a68d34da29bf7c8ccb5d355ba8b4babcad1f99798204e7abb43e54ee3d")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
	})
}

func TestPushInstallBundle(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		ref := info.registryAddress + "/test/push-bundle"

		// render the app to a bundle, we use the app from the push pull test above.
		build(t, cmd, dockerCli, "a-simple-app:1.0.0", filepath.Join("testdata", "push-pull"))

		// push it and install to check it is available
		t.Run("push-bundle", func(t *testing.T) {
			name := strings.Replace(t.Name(), "/", "_", 1)
			cmd.Command = dockerCli.Command("app", "image", "tag", "a-simple-app:1.0.0", ref)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			cmd.Command = dockerCli.Command("app", "push", ref)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			cmd.Command = dockerCli.Command("app", "run", ref, "--name", name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			cmd.Command = dockerCli.Command("service", "ls")
			assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), name))

			// ensure it doesn't confuse the next test
			cmd.Command = dockerCli.Command("app", "rm", name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			cmd.Command = dockerCli.Command("service", "ls")
			assert.Check(t, !strings.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), name))
		})

		// push it again using the first ref and install from the new ref to check it is also available
		t.Run("push-ref", func(t *testing.T) {
			name := strings.Replace(t.Name(), "/", "_", 1)
			ref2 := info.registryAddress + "/test/push-ref"
			cmd.Command = dockerCli.Command("app", "image", "tag", ref+":latest", ref2)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			cmd.Command = dockerCli.Command("app", "push", ref2)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			cmd.Command = dockerCli.Command("app", "run", ref2, "--name", name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			cmd.Command = dockerCli.Command("service", "ls")
			assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), name))
		})

		// push it again using an app pre-bundled and tagged in the bundle store and install it to check it is also available
		t.Run("push-bundleref", func(t *testing.T) {
			name := strings.Replace(t.Name(), "/", "_", 1)
			ref2 := ref + ":v0.42"
			// Create a new command so the bundle store can be trashed before installing the app
			cmdIsolatedStore, cleanupIsolatedStore := dockerCli.createTestCmd()

			// Enter the same context as `cmd` to run commands within the same environment
			cmdIsolatedStore.Command = dockerCli.Command("context", "create", "swarm-context", "--docker", fmt.Sprintf(`"host=tcp://%s"`, info.swarmAddress))
			icmd.RunCmd(cmdIsolatedStore).Assert(t, icmd.Success)
			cmdIsolatedStore.Env = append(cmdIsolatedStore.Env, "DOCKER_CONTEXT=swarm-context")

			// bundle the app again but this time with a tag to store it into the bundle store
			build(t, cmdIsolatedStore, dockerCli, ref2, filepath.Join("testdata", "push-pull"))
			// Push the app without tagging it explicitly
			cmdIsolatedStore.Command = dockerCli.Command("app", "push", ref2)
			icmd.RunCmd(cmdIsolatedStore).Assert(t, icmd.Success)
			// remove the bundle from the bundle store to be sure it won't be used instead of registry
			cleanupIsolatedStore()
			// install from the registry
			cmd.Command = dockerCli.Command("app", "run", ref2, "--name", name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			cmd.Command = dockerCli.Command("service", "ls")
			assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), name))
		})
	})
}
