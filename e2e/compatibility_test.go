package e2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	pollTimeout = 30 * time.Second
)

func runAppE2E(t *testing.T, info OrchestratorAndRegistryInfo, expectedOutput map[string][]string) {
	appName := "app-e2e"
	appImage := filepath.Join(info.registryAddress, "app-e2e:v0.9.0")
	imgPrefix := filepath.Join(info.registryAddress, "app-e2e")
	data, err := ioutil.ReadFile(filepath.Join("testdata", "compatibility", "bundle-v0.9.0.json"))
	assert.NilError(t, err)

	// update bundle
	bundleDir := filepath.Join(info.configDir, "app", "bundles", "docker.io", "library", "app-e2e", "_tags", "v0.9.0")

	assert.NilError(t, os.MkdirAll(bundleDir, os.FileMode(0777)))
	assert.NilError(t, ioutil.WriteFile(filepath.Join(bundleDir, "bundle.json"), data, os.FileMode(0644)))

	// load images build with an old Docker App version
	assert.NilError(t, info.LoadImageFromWeb("app-e2e:0.1.0-invoc", "https://github.com/docker/app-e2e/raw/master/images/v0.9.0/app-e2e-invoc.tar"))
	assert.NilError(t, info.LoadImageFromWeb("app-e2e/backend", "https://github.com/docker/app-e2e/raw/master/images/v0.9.0/backend.tar"))
	assert.NilError(t, info.LoadImageFromWeb("app-e2e/frontend", "https://github.com/docker/app-e2e/raw/master/images/v0.9.0/frontend.tar"))

	// tag app image
	info.dockerCmd("app", "image", "tag", "app-e2e:v0.9.0", appImage)
	// list images
	output := info.dockerCmd("app", "image", "ls")
	checkContains(t, output, []string{appName})
	// push app to registry
	info.dockerCmd("app", "push", appImage)
	// pull app from registry
	info.dockerCmd("app", "pull", appImage)
	// inspect bundle
	output = info.dockerCmd("app", "image", "inspect", appImage, "--pretty")
	checkContains(t, output,
		[]string{
			`name:\s+app-e2e`,
			fmt.Sprintf(`backend\s+1\s+.+%s`, imgPrefix),
			fmt.Sprintf(`frontend\s+1\s+8080\s+.+%s`, imgPrefix),
			`ports.frontend\s+8080`,
		})

	// render bundle
	output = info.dockerCmd("app", "image", "render", appImage)
	checkContains(t, output,
		[]string{
			fmt.Sprintf(`image:.+%s`, imgPrefix),
			fmt.Sprintf(`image:.+%s`, imgPrefix),
			"published: 8080",
			"target: 80",
		})

	// Install app
	output = info.dockerCmd("app", "run", appImage, "--name", appName)
	checkContains(t, output, expectedOutput["run"])

	// Status check -- poll app list
	checkStatus := func(lastAction string) {
		err = wait.Poll(2*time.Second, pollTimeout, func() (bool, error) {
			output = info.dockerCmd("app", "ls")
			fmt.Println(output)
			expectedLines := []string{
				`RUNNING APP\s+APP NAME\s+SERVICES\s+LAST ACTION\s+RESULT\s+CREATED\s+MODIFIED\s+REFERENCE`,
				fmt.Sprintf(`%s\s+%s \(0.1.0\)\s+.+%s\s+success\s+.+second[s]?\sago\s+.+second[s]?\sago\s+`, appName, appName, lastAction),
			}
			matches := true
			for _, expected := range expectedLines {
				exp := regexp.MustCompile(expected)
				matches = matches && exp.MatchString(output)
			}
			return matches, nil
		})
		assert.NilError(t, err)
	}

	// Inspect app
	output = info.dockerCmd("app", "inspect", appName, "--pretty")
	checkContains(t, output,
		[]string{
			"Running App:",
			fmt.Sprintf("Name: %s", appName),
			"Result: success",
			`ports.frontend: "8080"`,
		})

	// Update the application, changing the port
	output = info.dockerCmd("app", "update", appName, "--set", "ports.frontend=8081")
	checkContains(t, output, expectedOutput["upgrade"])

	// check status on upgrade
	checkStatus("upgrade")

	// Uninstall the application
	output = info.dockerCmd("app", "rm", appName)
	checkContains(t, output, expectedOutput["rm"])
}

func TestBackwardsCompatibilityV1(t *testing.T) {
	t.Run("Swarm", func(t *testing.T) {
		runWithDindSwarmAndRegistry(t, func(info OrchestratorAndRegistryInfo) {
			//expected_outputs
			expectedOutput := map[string][]string{
				"run": {
					`Creating service app-e2e_backend`,
					`Creating service app-e2e_frontend`,
					`Creating network app-e2e_default`,
				},
				"upgrade": {
					`Updating service app-e2e_backend`,
					`Updating service app-e2e_frontend`,
				},
				"rm": {
					`Removing service app-e2e_backend`,
					`Removing service app-e2e_frontend`,
					`Removing network app-e2e_default`,
				}}
			runAppE2E(t, info, expectedOutput)
		})
	})
	t.Run("Kubernetes", func(t *testing.T) {
		runWithKindAndRegistry(t, func(info OrchestratorAndRegistryInfo) {
			//expected_outputs
			expectedOutput := map[string][]string{
				"run": {
					`backend: Ready`,
					`frontend: Ready`,
					`Stack app-e2e is stable and running`,
				},
				"upgrade": {
					`Waiting for the stack to be stable and running...`,
					`backend: Ready`,
					`frontend: Ready`,
				},
				"rm": {},
			}
			runAppE2E(t, info, expectedOutput)
		})
	})
}
