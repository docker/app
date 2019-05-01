package e2e

import (
	"fmt"
	"math/rand"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/app/internal"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
	"gotest.tools/fs"
	"gotest.tools/icmd"
)

type dindSwarmAndRegistryInfo struct {
	swarmAddress    string
	registryAddress string
	configuredCmd   icmd.Cmd
	stopRegistry    func()
}

func runWithDindSwarmAndRegistry(t *testing.T, todo func(dindSwarmAndRegistryInfo)) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	registryPort := findAvailablePort()
	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()

	cmd.Env = append(cmd.Env, "DOCKER_TARGET_CONTEXT=swarm-target-context")

	// The dind doesn't have the cnab-app-base image so we save it in order to load it later
	saveCmd := icmd.Cmd{Command: dockerCli.Command("save", fmt.Sprintf("docker/cnab-app-base:%s", internal.Version), "-o", tmpDir.Join("cnab-app-base.tar.gz"))}
	icmd.RunCmd(saveCmd).Assert(t, icmd.Success)

	// we have a difficult constraint here:
	// - the registry must be reachable from the client side (for cnab-to-oci, which does not use the docker daemon to access the registry)
	// - the registry must be reachable from the dind daemon on the same address/port
	// Solution found is: fix the port of the registry to be the same internally and externally
	// and run the dind container in the same network namespace: this way 127.0.0.1:<registry-port> both resolves to the registry from the client and from dind

	swarm := NewContainer("docker:18.09-dind", 2375, "--insecure-registry", fmt.Sprintf("127.0.0.1:%d", registryPort))
	swarm.Start(t, "--expose", strconv.FormatInt(int64(registryPort), 10),
		"-p", fmt.Sprintf("%d:%d", registryPort, registryPort),
		"-p", "2375")
	defer swarm.Stop(t)

	registry := NewContainer("registry:2", registryPort)
	registry.StartWithContainerNetwork(t, swarm, "-e", "REGISTRY_VALIDATION_MANIFESTS_URLS_ALLOW=[^http]",
		"-e", fmt.Sprintf("REGISTRY_HTTP_ADDR=0.0.0.0:%d", registryPort))
	defer registry.StopNoFail()

	// We  need two contexts:
	// - one for `docker` so that it connects to the dind swarm created before
	// - the target context for the invocation image to install within the swarm
	cmd.Command = dockerCli.Command("context", "create", "swarm-context", "--docker", fmt.Sprintf(`"host=tcp://%s"`, swarm.GetAddress(t)), "--default-stack-orchestrator", "swarm")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// When creating a context on a Windows host we cannot use
	// the unix socket but it's needed inside the invocation image.
	// The workaround is to create a context with an empty host.
	// This host will default to the unix socket inside the
	// invocation image
	cmd.Command = dockerCli.Command("context", "create", "swarm-target-context", "--docker", "host=", "--default-stack-orchestrator", "swarm")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// Initialize the swarm
	cmd.Env = append(cmd.Env, "DOCKER_CONTEXT=swarm-context")
	cmd.Command = dockerCli.Command("swarm", "init")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
	// Load the needed base cnab image into the swarm docker engine
	cmd.Command = dockerCli.Command("load", "-i", tmpDir.Join("cnab-app-base.tar.gz"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	info := dindSwarmAndRegistryInfo{
		configuredCmd:   cmd,
		registryAddress: registry.GetAddress(t),
		swarmAddress:    swarm.GetAddress(t),
		stopRegistry:    registry.StopNoFail,
	}
	todo(info)

}

func TestPushInstall(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		ref := info.registryAddress + "/test/push-pull"
		cmd.Command = dockerCli.Command("app", "push", "--tag", ref, "--insecure-registries="+info.registryAddress, filepath.Join("testdata", "push-pull", "push-pull.dockerapp"))
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "install", "--insecure-registries="+info.registryAddress, ref, "--name", t.Name())
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		cmd.Command = dockerCli.Command("service", "ls")
		assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), ref))
	})
}

func TestPushPullInstall(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		ref := info.registryAddress + "/test/push-pull"
		tag := ":v.0.0.1"
		cmd.Command = dockerCli.Command("app", "push", "--tag", ref+tag, "--insecure-registries="+info.registryAddress, filepath.Join("testdata", "push-pull", "push-pull.dockerapp"))
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		cmd.Command = dockerCli.Command("app", "pull", ref+tag, "--insecure-registries="+info.registryAddress)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// stop the registry
		info.stopRegistry()

		// install without --pull should succeed (rely on local store)
		cmd.Command = dockerCli.Command("app", "install", "--insecure-registries="+info.registryAddress, ref+tag, "--name", t.Name())
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		cmd.Command = dockerCli.Command("service", "ls")
		assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), ref))

		// listing the installed application shows the pulled application reference
		cmd.Command = dockerCli.Command("app", "list")
		checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
			[]string{
				fmt.Sprintf(`%s\s+push-pull \(1.1.0-beta1\)\s+install\s+success\s+.+second[s]?\s+.+second[s]?\s+%s`, t.Name(), ref+tag),
			})

		// install with --pull should fail (registry is stopped)
		cmd.Command = dockerCli.Command("app", "install", "--pull", "--insecure-registries="+info.registryAddress, ref, "--name", t.Name()+"2")
		assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 1}).Combined(), "failed to resolve bundle manifest"))
	})
}

func TestPushInstallBundle(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		ref := info.registryAddress + "/test/push-bundle"

		tmpDir := fs.NewDir(t, t.Name())
		defer tmpDir.Remove()
		bundleFile := tmpDir.Join("bundle.json")

		// render the app to a bundle, we use the app from the push pull test above.
		cmd.Command = dockerCli.Command("app", "bundle", "-o", bundleFile, filepath.Join("testdata", "push-pull", "push-pull.dockerapp"))
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// push it and install to check it is available
		t.Run("push-bundle", func(t *testing.T) {
			name := strings.Replace(t.Name(), "/", "_", 1)
			cmd.Command = dockerCli.Command("app", "push", "--insecure-registries="+info.registryAddress, "--tag", ref, bundleFile)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			cmd.Command = dockerCli.Command("app", "install", "--insecure-registries="+info.registryAddress, ref, "--name", name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			cmd.Command = dockerCli.Command("service", "ls")
			assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), ref))

			// ensure it doesn't confuse the next test
			cmd.Command = dockerCli.Command("app", "uninstall", name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			cmd.Command = dockerCli.Command("service", "ls")
			assert.Check(t, !strings.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), ref))
		})

		// push it again using the first ref and install from the new ref to check it is also available
		t.Run("push-ref", func(t *testing.T) {
			name := strings.Replace(t.Name(), "/", "_", 1)
			ref2 := info.registryAddress + "/test/push-ref"
			cmd.Command = dockerCli.Command("app", "push", "--insecure-registries="+info.registryAddress, "--tag", ref2, ref+":latest")
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			cmd.Command = dockerCli.Command("app", "install", "--insecure-registries="+info.registryAddress, ref2, "--name", name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			cmd.Command = dockerCli.Command("service", "ls")
			assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), ref2))
		})
	})
}

func findAvailablePort() int {
	rand.Seed(time.Now().UnixNano())
	for {
		candidate := (rand.Int() % 2000) + 5000
		if isPortAvailable(candidate) {
			return candidate
		}
	}
}

func isPortAvailable(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	defer l.Close()
	return true
}
