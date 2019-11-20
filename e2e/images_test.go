package e2e

import (
	"bufio"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/icmd"
)

func insertBundles(t *testing.T, cmd icmd.Cmd) {
	// Push an application so that we can later pull it by digest
	cmd.Command = dockerCli.Command("app", "build", "--no-resolve-image", "--tag", "my.registry:5000/c-myapp", filepath.Join("testdata", "push-pull"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
	cmd.Command = dockerCli.Command("app", "build", "--no-resolve-image", "--tag", "b-simple-app", filepath.Join("testdata", "simple"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
	cmd.Command = dockerCli.Command("app", "build", "--no-resolve-image", "--tag", "a-simple-app", filepath.Join("testdata", "simple"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
}

func assertImageListOutput(t *testing.T, cmd icmd.Cmd, expected string) {
	result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
	stdout := result.Stdout()
	match, err := regexp.MatchString(expected, stdout)
	assert.NilError(t, err)
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

func verifyImageIDListOutput(t *testing.T, cmd icmd.Cmd, expectedCount int) {
	cmd.Command = dockerCli.Command("app", "image", "ls", "-q")
	result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
	scanner := bufio.NewScanner(strings.NewReader(result.Stdout()))
	count := 0
	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		assert.Error(t, err, "Verification failed")
	}
	assert.Equal(t, expectedCount, count)
}

func TestImageList(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		insertBundles(t, cmd)

		expected := `REPOSITORY                 TAG                 APP IMAGE ID        APP NAME            CREATED                  
a-simple-app               latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
b-simple-app               latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
my.registry:5000/c-myapp   latest              [a-f0-9]{12}        push-pull           [La-z0-9 ]+ ago[ ]*
`

		expectImageListOutput(t, cmd, expected)
	})
}

func TestImageListQuiet(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		insertBundles(t, cmd)
		verifyImageIDListOutput(t, cmd, 3)
	})
}

func TestImageListDigests(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		insertBundles(t, cmd)
		expected := `REPOSITORY                 TAG                 DIGEST              APP IMAGE ID        APP NAME                                CREATED                  
a-simple-app               latest              <none>              [a-f0-9]{12}        simple                                  [La-z0-9 ]+ ago[ ]*
b-simple-app               latest              <none>              [a-f0-9]{12}        simple                                  [La-z0-9 ]+ ago[ ]*
my.registry:5000/c-myapp   latest              <none>              [a-f0-9]{12}        push-pull                               [La-z0-9 ]+ ago[ ]*
`
		expectImageListDigestsOutput(t, cmd, expected)
	})
}

func TestImageRm(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd

		insertBundles(t, cmd)

		cmd.Command = dockerCli.Command("app", "image", "rm", "my.registry:5000/c-myapp:latest")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 0,
			Out:      "Deleted: my.registry:5000/c-myapp:latest",
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

		expectedOutput := "REPOSITORY          TAG                 APP IMAGE ID        APP NAME            CREATED             \n"
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

		singleImageExpectation := `REPOSITORY          TAG                 APP IMAGE ID        APP NAME            CREATED[ ]*
a-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
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
		expectImageListOutput(t, cmd, `REPOSITORY          TAG                 APP IMAGE ID        APP NAME            CREATED[ ]*
a-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
b-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
`)

		// target tag
		dockerAppImageTag("a-simple-app", "a-simple-app:0.1")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `REPOSITORY          TAG                 APP IMAGE ID        APP NAME            CREATED[ ]*
a-simple-app        0.1                 [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
a-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
b-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
`)

		// source tag
		dockerAppImageTag("a-simple-app:0.1", "c-simple-app")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `REPOSITORY          TAG                 APP IMAGE ID        APP NAME            CREATED[ ]*
a-simple-app        0.1                 [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
a-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
b-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
c-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
`)

		// source and target tags
		dockerAppImageTag("a-simple-app:0.1", "b-simple-app:0.2")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `REPOSITORY          TAG                 APP IMAGE ID        APP NAME            CREATED[ ]*
a-simple-app        0.1                 [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
a-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
b-simple-app        0.2                 [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
b-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
c-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
`)

		// given a new application
		cmd.Command = dockerCli.Command("app", "build", "--tag", "push-pull", filepath.Join("testdata", "push-pull"))
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `REPOSITORY          TAG                 APP IMAGE ID        APP NAME            CREATED[ ]*
a-simple-app        0.1                 [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
a-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
b-simple-app        0.2                 [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
b-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
c-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
push-pull           latest              [a-f0-9]{12}        push-pull           [La-z0-9 ]+ ago[ ]*
`)

		// can be tagged to an existing tag
		dockerAppImageTag("push-pull", "b-simple-app:0.2")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `REPOSITORY          TAG                 APP IMAGE ID        APP NAME            CREATED[ ]*
a-simple-app        0.1                 [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
a-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
b-simple-app        0.2                 [a-f0-9]{12}        push-pull           [La-z0-9 ]+ ago[ ]*
b-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
c-simple-app        latest              [a-f0-9]{12}        simple              [La-z0-9 ]+ ago[ ]*
push-pull           latest              [a-f0-9]{12}        push-pull           [La-z0-9 ]+ ago[ ]*
`)
	})
}
