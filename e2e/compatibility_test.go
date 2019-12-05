package e2e

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/util/wait"

	"gotest.tools/fs"
)

const (
	pollTimeout = 30 * time.Second
)

func loadAndTagImage(info dindSwarmAndRegistryInfo, tmpDir *fs.Dir, tag string, url string) error {
	err := downloadImageTarball(tmpDir.Join("image.tar"), url)
	if err != nil {
		return err
	}

	digest := ""
	combined := info.dockerCmd("load", "-q", "-i", tmpDir.Join("image.tar"))
	for _, line := range strings.Split(combined, "\n") {
		if strings.Contains(line, "sha256:") {
			digest = strings.Split(line, "sha256:")[1]
		}
	}
	if digest == "" {
		return errors.New("Image digest not found in docker load's stdout")
	}

	digest = strings.Trim(digest, " \r\n")
	info.dockerCmd("tag", digest, tag)

	return nil
}

func downloadImageTarball(filepath string, url string) error {
	client := http.Client{Timeout: time.Minute * 1}
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	// Write the body to file
	_, err = io.Copy(out, res.Body)
	return err
}

func TestBackwardsCompatibilityV1(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		appName := "app-e2e"

		data, err := ioutil.ReadFile(filepath.Join("testdata", "compatibility", "bundle-v0.9.0.json"))
		assert.NilError(t, err)
		// update bundle
		bundleDir := filepath.Join(info.configDir, "app", "bundles", "docker.io", "library", "app-e2e", "_tags", "v0.9.0")
		assert.NilError(t, os.MkdirAll(bundleDir, os.FileMode(0777)))
		assert.NilError(t, ioutil.WriteFile(filepath.Join(bundleDir, "bundle.json"), data, os.FileMode(0644)))

		// load images build with an old Docker App version
		assert.NilError(t, loadAndTagImage(info, info.tmpDir, "app-e2e:0.1.0-invoc", "https://github.com/docker/app-e2e/raw/master/images/v0.9.0/app-e2e-invoc.tar"))
		assert.NilError(t, loadAndTagImage(info, info.tmpDir, "app-e2e/backend", "https://github.com/docker/app-e2e/raw/master/images/v0.9.0/backend.tar"))
		assert.NilError(t, loadAndTagImage(info, info.tmpDir, "app-e2e/frontend", "https://github.com/docker/app-e2e/raw/master/images/v0.9.0/frontend.tar"))

		// list images
		output := info.dockerCmd("app", "image", "ls")
		checkContains(t, output, []string{appName})
		// inspect bundle
		output = info.dockerCmd("app", "image", "inspect", "app-e2e:v0.9.0", "--pretty")
		checkContains(t, output,
			[]string{
				`name:\s+app-e2e`,
				`backend\s+1\s+app-e2e/backend`,
				`frontend\s+1\s+8080\s+app-e2e/frontend`,
				`ports.frontend\s+8080`,
			})

		// render bundle
		output = info.dockerCmd("app", "image", "render", "app-e2e:v0.9.0")
		checkContains(t, output,
			[]string{
				"image: app-e2e/frontend",
				"image: app-e2e/backend",
				"published: 8080",
				"target: 80",
			})

		// Install app
		output = info.dockerCmd("app", "run", "app-e2e:v0.9.0", "--name", appName)
		checkContains(t, output,
			[]string{
				fmt.Sprintf("Creating service %s_backend", appName),
				fmt.Sprintf("Creating service %s_frontend", appName),
				fmt.Sprintf("Creating network %s_default", appName),
			})

		// Status check -- poll app list
		checkStatus := func(lastAction string) {
			err = wait.Poll(2*time.Second, pollTimeout, func() (bool, error) {
				output = info.dockerCmd("app", "ls")
				expectedLines := []string{
					`RUNNING APP\s+APP NAME\s+SERVICES\s+LAST ACTION\s+RESULT\s+CREATED\s+MODIFIED\s+REFERENCE`,
					fmt.Sprintf(`%s\s+%s \(0.1.0\)\s+2/2\s+%s\s+success\s+.+second[s]?\sago\s+.+second[s]?\sago\s+`, appName, appName, lastAction),
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

		queryService := func(port string) {
			err = wait.Poll(2*time.Second, pollTimeout, func() (bool, error) {
				// Check the frontend service responds
				url := `http://localhost:` + port
				output = info.execCmd("/usr/bin/wget", "-O", "-", url)
				return strings.Contains(output, `Hi there, I love Docker!`), nil
			})
			output = ""
			if err != nil {
				output = info.dockerCmd("stack", "ps", appName)
			}
			assert.NilError(t, err, output)
		}

		// Check status on install
		checkStatus("install")

		// query deployed service
		queryService("8080")

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
		checkContains(t, output,
			[]string{
				fmt.Sprintf("Updating service %s_backend", appName),
				fmt.Sprintf("Updating service %s_frontend", appName),
			})

		// check status on upgrade
		checkStatus("upgrade")

		// Check the frontend service responds on the new port
		queryService("8081")

		// Uninstall the application
		output = info.dockerCmd("app", "rm", appName)
		checkContains(t, output,
			[]string{
				fmt.Sprintf("Removing service %s_backend", appName),
				fmt.Sprintf("Removing service %s_frontend", appName),
				fmt.Sprintf("Removing network %s_default", appName),
			})
	})
}
