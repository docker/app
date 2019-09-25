package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/app/internal"
	"github.com/docker/cnab-to-oci/converter"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
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
	registryLogs    func() string
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
		registryLogs:    registry.Logs(t),
	}
	todo(info)

}

func TestPushArchs(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		testCases := []struct {
			name              string
			args              []string
			expectedPlatforms []manifestlist.PlatformSpec
		}{
			{
				name: "default",
				args: []string{},
				expectedPlatforms: []manifestlist.PlatformSpec{
					{
						OS:           "linux",
						Architecture: "amd64",
					},
				},
			},
			{
				name: "all-platforms",
				args: []string{"--all-platforms"},
				expectedPlatforms: []manifestlist.PlatformSpec{
					{
						OS:           "linux",
						Architecture: "amd64",
					},
					{
						OS:           "linux",
						Architecture: "386",
					},
					{
						OS:           "linux",
						Architecture: "ppc64le",
					},
					{
						OS:           "linux",
						Architecture: "s390x",
					},
					{
						OS:           "linux",
						Architecture: "arm",
						Variant:      "v5",
					},
					{
						OS:           "linux",
						Architecture: "arm",
						Variant:      "v6",
					},
					{
						OS:           "linux",
						Architecture: "arm",
						Variant:      "v7",
					},
					{
						OS:           "linux",
						Architecture: "arm64",
						Variant:      "v8",
					},
				},
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				cmd := info.configuredCmd
				ref := info.registryAddress + "/test/push-pull:1"
				args := []string{"app", "push", "--tag", ref, "--insecure-registries=" + info.registryAddress}
				args = append(args, testCase.args...)
				args = append(args, filepath.Join("testdata", "push-pull", "push-pull.dockerapp"))
				cmd.Command = dockerCli.Command(args...)
				icmd.RunCmd(cmd).Assert(t, icmd.Success)

				var index v1.Index
				headers := map[string]string{
					"Accept": "application/vnd.docker.distribution.manifest.list.v2+json",
				}
				err := httpGet("http://"+info.registryAddress+"/v2/test/push-pull/manifests/1", headers, &index)
				assert.NilError(t, err, info.registryLogs())
				digest, err := getManifestListDigest(index)
				assert.NilError(t, err, info.registryLogs())
				var manifestList manifestlist.ManifestList
				err = httpGet("http://"+info.registryAddress+"/v2/test/push-pull/manifests/"+digest.String(), headers, &manifestList)
				assert.NilError(t, err)
				assert.Equal(t, len(manifestList.Manifests), len(testCase.expectedPlatforms), "Unexpected number of platforms")
				for _, m := range manifestList.Manifests {
					assert.Assert(t, cmp.Contains(testCase.expectedPlatforms, m.Platform), "Platform expected but not found: %s", m.Platform)
				}
			})
		}
	})
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
		cmd.Command = dockerCli.Command("app", "ls")
		checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
			[]string{
				fmt.Sprintf(`%s\s+push-pull \(1.1.0-beta1\)\s+install\s+success\s+.+second[s]?\sago\s+.+second[s]?\sago\s+%s`, t.Name(), ref+tag),
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
			cmd.Command = dockerCli.Command("app", "rm", name)
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

		// push it again using an app pre-bundled and tagged in the bundle store and install it to check it is also available
		t.Run("push-bundleref", func(t *testing.T) {
			name := strings.Replace(t.Name(), "/", "_", 1)
			ref2 := ref + ":v0.42"
			// Create a new command so the bundle store can be trashed before installing the app
			cmd2, cleanup2 := dockerCli.createTestCmd()
			// bundle the app again but this time with a tag to store it into the bundle store
			cmd2.Command = dockerCli.Command("app", "bundle", "--tag", ref2, "-o", bundleFile, filepath.Join("testdata", "push-pull", "push-pull.dockerapp"))
			icmd.RunCmd(cmd2).Assert(t, icmd.Success)
			// Push the app without tagging it explicitly
			cmd2.Command = dockerCli.Command("app", "push", "--insecure-registries="+info.registryAddress, ref2)
			icmd.RunCmd(cmd2).Assert(t, icmd.Success)
			// remove the bundle from the bundle store to be sure it won't be used instead of registry
			cleanup2()
			// install from the registry
			cmd.Command = dockerCli.Command("app", "install", "--insecure-registries="+info.registryAddress, ref2, "--name", name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			cmd.Command = dockerCli.Command("service", "ls")
			assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), ref))
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

func httpGet(url string, headers map[string]string, obj interface{}) error {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	r, err := client.Do(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("unexpected http error code %d with message %s", r.StatusCode, string(body))
	}
	if err := json.NewDecoder(r.Body).Decode(obj); err != nil {
		return err
	}
	return nil
}

func getManifestListDigest(index v1.Index) (digest.Digest, error) {
	for _, m := range index.Manifests {
		if m.Annotations[converter.CNABDescriptorTypeAnnotation] == "component" {
			return m.Digest, nil
		}
	}
	return "", fmt.Errorf("Service image not found")
}
