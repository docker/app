package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/docker/cnab-to-oci/converter"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"
	"gotest.tools/icmd"
)

type dindSwarmAndRegistryInfo struct {
	swarmAddress    string
	registryAddress string
	configuredCmd   icmd.Cmd
	stopRegistry    func()
	registryLogs    func() string
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
				args := []string{"app", "push", "--tag", ref}
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

func TestPushInsecureRegistry(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		ref := info.registryAddress + "/test/push-insecure"

		// create a command outside of the dind context so without the insecure registry configured
		cmdNoInsecureRegistry, cleanupNoInsecureRegistryCommand := dockerCli.createTestCmd()
		defer cleanupNoInsecureRegistryCommand()
		cmdNoInsecureRegistry.Command = dockerCli.Command("app", "push", "--tag", ref, filepath.Join("testdata", "push-pull", "push-pull.dockerapp"))
		icmd.RunCmd(cmdNoInsecureRegistry).Assert(t, icmd.Expected{ExitCode: 1})

		// run the push with the command inside dind context configured to allow access to the insecure registry
		cmd := info.configuredCmd
		cmd.Command = dockerCli.Command("app", "push", "--tag", ref, filepath.Join("testdata", "push-pull", "push-pull.dockerapp"))
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
	})
}

func TestPushInstall(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		ref := info.registryAddress + "/test/push-pull"
		cmd.Command = dockerCli.Command("app", "push", "--tag", ref, filepath.Join("testdata", "push-pull", "push-pull.dockerapp"))
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "install", ref, "--name", t.Name())
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
		cmd.Command = dockerCli.Command("app", "push", "--tag", ref+tag, filepath.Join("testdata", "push-pull", "push-pull.dockerapp"))
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		cmd.Command = dockerCli.Command("app", "pull", ref+tag)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// stop the registry
		info.stopRegistry()

		// install from local store
		cmd.Command = dockerCli.Command("app", "install", ref+tag, "--name", t.Name())
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		cmd.Command = dockerCli.Command("service", "ls")
		assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), ref))

		// listing the installed application shows the pulled application reference
		cmd.Command = dockerCli.Command("app", "ls")
		checkContains(t, icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(),
			[]string{
				fmt.Sprintf(`%s\s+push-pull \(1.1.0-beta1\)\s+install\s+success\s+.+second[s]?\sago\s+.+second[s]?\sago\s+%s`, t.Name(), ref+tag),
			})

		// install should fail (registry is stopped)
		cmd.Command = dockerCli.Command("app", "install", "unknown")
		//nolint: lll
		expected := `Unable to find application image "unknown:latest" locally
Unable to find application "unknown": failed to resolve bundle manifest "docker.io/library/unknown:latest": pull access denied, repository does not exist or may require authorization: server message: insufficient_scope: authorization failed`
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      expected,
		})
	})
}

func TestPushInstallBundle(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		ref := info.registryAddress + "/test/push-bundle"

		// render the app to a bundle, we use the app from the push pull test above.
		cmd.Command = dockerCli.Command("app", "build", filepath.Join("testdata", "push-pull", "push-pull.dockerapp"), "a-simple-app:1.0.0")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// push it and install to check it is available
		t.Run("push-bundle", func(t *testing.T) {
			name := strings.Replace(t.Name(), "/", "_", 1)
			cmd.Command = dockerCli.Command("app", "push", "--tag", ref, "a-simple-app:1.0.0")
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			cmd.Command = dockerCli.Command("app", "install", ref, "--name", name)
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
			cmd.Command = dockerCli.Command("app", "push", "--tag", ref2, ref+":latest")
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			cmd.Command = dockerCli.Command("app", "install", ref2, "--name", name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			cmd.Command = dockerCli.Command("service", "ls")
			assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), ref2))
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
			cmdIsolatedStore.Command = dockerCli.Command("app", "build", filepath.Join("testdata", "push-pull", "push-pull.dockerapp"), ref2)
			icmd.RunCmd(cmdIsolatedStore).Assert(t, icmd.Success)
			// Push the app without tagging it explicitly
			cmdIsolatedStore.Command = dockerCli.Command("app", "push", ref2)
			icmd.RunCmd(cmdIsolatedStore).Assert(t, icmd.Success)
			// remove the bundle from the bundle store to be sure it won't be used instead of registry
			cleanupIsolatedStore()
			// install from the registry
			cmd.Command = dockerCli.Command("app", "install", ref2, "--name", name)
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
