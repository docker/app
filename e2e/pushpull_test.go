package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/cnab-to-oci/converter"
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

func TestPushApplication(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		ref := info.registryAddress + "/c-myapp:0.1.0"
		insertBundles(t, cmd, info)

		cmd.Command = dockerCli.Command("app", "image", "tag", info.registryAddress+"/c-myapp", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "push", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		var index v1.Index
		headers := map[string]string{
			"Accept": "application/vnd.docker.distribution.manifest.list.v2+json",
		}
		err := httpGet("http://"+info.registryAddress+"/v2/c-myapp/manifests/0.1.0", headers, &index)
		assert.NilError(t, err, info.registryLogs())

		t.Run("invocation-image", func(t *testing.T) {
			_, err := getInvocationImageDigest(index)
			assert.NilError(t, err, info.registryLogs)
		})

		t.Run("service-image", func(t *testing.T) {
			assert.Equal(t, countServiceImages(index), 2)
		})

		t.Run("config", func(t *testing.T) {
			assert.Assert(t, hasConfig(index))
		})
	})
}

func TestPushInsecureRegistry(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		insertBundles(t, info.configuredCmd, info)
		ref := info.registryAddress + "/c-myapp"

		// create a command outside of the dind context so without the insecure registry configured
		cmdNoInsecureRegistry, cleanupNoInsecureRegistryCommand := dockerCli.createTestCmd()
		defer cleanupNoInsecureRegistryCommand()
		cmdNoInsecureRegistry.Command = dockerCli.Command("app", "push", ref)
		icmd.RunCmd(cmdNoInsecureRegistry).Assert(t, icmd.Expected{ExitCode: 1})

		// run the push with the command inside dind context configured to allow access to the insecure registry
		cmd := info.configuredCmd
		cmd.Command = dockerCli.Command("app", "push", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
	})
}

func TestPushInstall(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		insertBundles(t, cmd, info)
		ref := info.registryAddress + "/c-myapp"
		cmd.Command = dockerCli.Command("app", "push", ref)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "install", ref, "--pull", "--name", t.Name())
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		cmd.Command = dockerCli.Command("service", "ls")
		assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), ref))
	})
}

func TestPushPullInstall(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		insertBundles(t, cmd, info)
		ref := info.registryAddress + "/c-myapp"

		// tag and push the image
		tag := ":v.0.0.1"
		cmd.Command = dockerCli.Command("app", "image", "tag", ref, ref+tag)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		cmd.Command = dockerCli.Command("app", "push", ref+tag)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// stop the registry
		info.stopRegistry()

		// install without --pull should succeed (rely on local store)
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

		// install with --pull should fail (registry is stopped)
		cmd.Command = dockerCli.Command("app", "install", "--pull", ref, "--name", t.Name()+"2")
		assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 1}).Combined(), "failed to resolve bundle manifest"))
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
			cmd.Command = dockerCli.Command("app", "image", "tag", "a-simple-app:1.0.0", ref)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			cmd.Command = dockerCli.Command("app", "push", ref)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			cmd.Command = dockerCli.Command("app", "install", ref, "--pull", "--name", name)
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
			cmd.Command = dockerCli.Command("app", "image", "tag", ref+":latest", ref2)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			cmd.Command = dockerCli.Command("app", "push", ref2)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)

			cmd.Command = dockerCli.Command("app", "install", ref2, "--pull", "--name", name)
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
			cmd.Command = dockerCli.Command("app", "install", ref2, "--pull", "--name", name)
			icmd.RunCmd(cmd).Assert(t, icmd.Success)
			cmd.Command = dockerCli.Command("service", "ls")
			assert.Check(t, cmp.Contains(icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined(), ref))
		})
	})
}

func TestAppDoesntExist(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		ref := info.registryAddress + "/c-myapp"
		insertBundles(t, cmd, info)

		cmd.Command = dockerCli.Command("app", "push", "unexisting_application")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `An application does not exist locally with name: "unexisting_application"`,
		})

		cmd.Command = dockerCli.Command("app", "push", ref+":unexisting_tag")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      fmt.Sprintf("An application does not exist locally with name: %q", ref+":unexisting_tag"),
		})
	})
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

func getInvocationImageDigest(index v1.Index) (digest.Digest, error) {
	for _, m := range index.Manifests {
		if m.Annotations[converter.CNABDescriptorTypeAnnotation] == "invocation" {
			return m.Digest, nil
		}
	}
	return "", fmt.Errorf("Invocation image not found")
}

func countServiceImages(index v1.Index) int {
	count := 0
	for _, m := range index.Manifests {
		if m.Annotations[converter.CNABDescriptorTypeAnnotation] == "component" {
			count++
		}
	}
	return count
}

func hasConfig(index v1.Index) bool {
	for _, m := range index.Manifests {
		if m.Annotations[converter.CNABDescriptorTypeAnnotation] == "config" {
			return true
		}
	}
	return false
}
