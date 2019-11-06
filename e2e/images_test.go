package e2e

import (
	"bufio"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/icmd"
)

func insertBundles(t *testing.T, cmd icmd.Cmd, info dindSwarmAndRegistryInfo) {
	// Push an application so that we can later pull it by digest
	cmd.Command = dockerCli.Command("app", "build", "--no-resolve-image", "--tag", info.registryAddress+"/c-myapp", filepath.Join("testdata", "push-pull"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
	cmd.Command = dockerCli.Command("app", "build", "--no-resolve-image", "--tag", "b-simple-app", filepath.Join("testdata", "simple"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
	cmd.Command = dockerCli.Command("app", "build", "--no-resolve-image", "--tag", "a-simple-app", filepath.Join("testdata", "simple"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
}

func assertImageListOutput(t *testing.T, cmd icmd.Cmd, expected string) {
	result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
	match, _ := regexp.MatchString(expected, result.Stdout())
	assert.Assert(t, match)
}

func expectImageListOutput(t *testing.T, cmd icmd.Cmd, output string) {
	cmd.Command = dockerCli.Command("app", "image", "ls")
	assertImageListOutput(t, cmd, output)
}

func expectImageListDigestsOutput(t *testing.T, cmd icmd.Cmd, output string) {
	cmd.Command = dockerCli.Command("app", "image", "ls", "--digests")
	assertImageListOutput(t, cmd, output)
}

func verifyImageIDListOutput(t *testing.T, cmd icmd.Cmd, count int, distinct int) {
	cmd.Command = dockerCli.Command("app", "image", "ls", "-q")
	result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
	scanner := bufio.NewScanner(strings.NewReader(result.Stdout()))
	lines := []string{}
	counter := make(map[string]int)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		counter[scanner.Text()]++
	}
	if err := scanner.Err(); err != nil {
		assert.Error(t, err, "Verification failed")
	}
	assert.Equal(t, len(lines), count)
	assert.Equal(t, len(counter), distinct)
}

func TestImageList(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		insertBundles(t, cmd, info)

		expected := `REPOSITORY             TAG    APP IMAGE ID APP NAME
%s latest [a-f0-9]{12} push-pull
a-simple-app           latest [a-f0-9]{12} simple
b-simple-app           latest [a-f0-9]{12} simple
`
		expectedOutput := fmt.Sprintf(expected, info.registryAddress+"/c-myapp")
		expectImageListOutput(t, cmd, expectedOutput)
	})
}

func TestImageListQuiet(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		insertBundles(t, cmd, info)
		verifyImageIDListOutput(t, cmd, 3, 2)
	})
}

func TestImageListDigests(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		insertBundles(t, cmd, info)
		expected := `REPOSITORY             TAG    DIGEST APP IMAGE ID APP NAME
%s latest <none> [a-f0-9]{12} push-pull
a-simple-app           latest <none> [a-f0-9]{12} simple
b-simple-app           latest <none> [a-f0-9]{12} simple
`
		expectedOutput := fmt.Sprintf(expected, info.registryAddress+"/c-myapp")
		expectImageListDigestsOutput(t, cmd, expectedOutput)
	})
}

func TestImageRm(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		insertBundles(t, cmd, info)

		cmd.Command = dockerCli.Command("app", "image", "rm", info.registryAddress+"/c-myapp:latest")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "Deleted: " + info.registryAddress + "/c-myapp:latest",
		})

		cmd.Command = dockerCli.Command("app", "image", "rm", "a-simple-app", "b-simple-app:latest")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 0,
			Out: `Deleted: a-simple-app:latest
Deleted: b-simple-app:latest`,
		})

		cmd.Command = dockerCli.Command("app", "image", "rm", "b-simple-app")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `b-simple-app:latest: reference not found`,
		})

		expectedOutput := "REPOSITORY TAG APP IMAGE ID APP NAME\n"
		expectImageListOutput(t, cmd, expectedOutput)
	})
}

func TestImageTag(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		dockerAppImageTag := func(args ...string) {
			cmdArgs := append([]string{"app", "image", "tag"}, args...)
			cmd.Command = dockerCli.Command(cmdArgs...)
		}

		// given a first available image
		cmd.Command = dockerCli.Command("app", "build", "--tag", "a-simple-app", filepath.Join("testdata", "simple"))
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		singleImageExpectation := `REPOSITORY   TAG    APP IMAGE ID APP NAME
a-simple-app latest [a-f0-9]{12} simple
`
		expectImageListOutput(t, cmd, singleImageExpectation)

		// with no argument
		dockerAppImageTag()
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `"docker app image tag" requires exactly 2 arguments.`,
		})

		// with one argument
		dockerAppImageTag("a-simple-app")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `"docker app image tag" requires exactly 2 arguments.`,
		})

		// with invalid src reference
		dockerAppImageTag("a-simple-app$2", "b-simple-app")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `could not parse "a-simple-app$2" as a valid reference`,
		})

		// with invalid target reference
		dockerAppImageTag("a-simple-app", "b@simple-app")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `could not parse "b@simple-app" as a valid reference`,
		})

		// with unexisting source image
		dockerAppImageTag("b-simple-app", "target")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `could not tag "b-simple-app": no such App image`,
		})

		// with unexisting source tag
		dockerAppImageTag("a-simple-app:not-a-tag", "target")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `could not tag "a-simple-app:not-a-tag": no such App image`,
		})

		// tag image with only names
		dockerAppImageTag("a-simple-app", "b-simple-app")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `REPOSITORY   TAG    APP IMAGE ID APP NAME
a-simple-app latest [a-f0-9]{12} simple
b-simple-app latest [a-f0-9]{12} simple
`)

		// target tag
		dockerAppImageTag("a-simple-app", "a-simple-app:0.1")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `REPOSITORY   TAG    APP IMAGE ID APP NAME
a-simple-app 0.1    [a-f0-9]{12} simple
a-simple-app latest [a-f0-9]{12} simple
b-simple-app latest [a-f0-9]{12} simple
`)

		// source tag
		dockerAppImageTag("a-simple-app:0.1", "c-simple-app")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `REPOSITORY   TAG    APP IMAGE ID APP NAME
a-simple-app 0.1    [a-f0-9]{12} simple
a-simple-app latest [a-f0-9]{12} simple
b-simple-app latest [a-f0-9]{12} simple
c-simple-app latest [a-f0-9]{12} simple
`)

		// source and target tags
		dockerAppImageTag("a-simple-app:0.1", "b-simple-app:0.2")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `REPOSITORY   TAG    APP IMAGE ID APP NAME
a-simple-app 0.1    [a-f0-9]{12} simple
a-simple-app latest [a-f0-9]{12} simple
b-simple-app 0.2    [a-f0-9]{12} simple
b-simple-app latest [a-f0-9]{12} simple
c-simple-app latest [a-f0-9]{12} simple
`)

		// given a new application
		cmd.Command = dockerCli.Command("app", "build", "--tag", "push-pull", filepath.Join("testdata", "push-pull"))
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `REPOSITORY   TAG    APP IMAGE ID APP NAME
a-simple-app 0.1    [a-f0-9]{12} simple
a-simple-app latest [a-f0-9]{12} simple
b-simple-app 0.2    [a-f0-9]{12} simple
b-simple-app latest [a-f0-9]{12} simple
c-simple-app latest [a-f0-9]{12} simple
push-pull    latest [a-f0-9]{12} push-pull
`)

		// can be tagged to an existing tag
		dockerAppImageTag("push-pull", "b-simple-app:0.2")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `REPOSITORY   TAG    APP IMAGE ID APP NAME
a-simple-app 0.1    [a-f0-9]{12} simple
a-simple-app latest [a-f0-9]{12} simple
b-simple-app 0.2    [a-f0-9]{12} push-pull
b-simple-app latest [a-f0-9]{12} simple
c-simple-app latest [a-f0-9]{12} simple
push-pull    latest [a-f0-9]{12} push-pull
`)
	})
}
