package e2e

import (
	"fmt"
	"path/filepath"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/icmd"
)

func insertBundles(t *testing.T, cmd icmd.Cmd, info dindSwarmAndRegistryInfo) {
	// Push an application so that we can later pull it by digest
	cmd.Command = dockerCli.Command("app", "build", filepath.Join("testdata", "push-pull", "push-pull.dockerapp"), info.registryAddress+"/c-myapp")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
	cmd.Command = dockerCli.Command("app", "build", filepath.Join("testdata", "simple", "simple.dockerapp"), "b-simple-app")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
	cmd.Command = dockerCli.Command("app", "build", filepath.Join("testdata", "simple", "simple.dockerapp"), "a-simple-app")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
}

func expectImageListOutput(t *testing.T, cmd icmd.Cmd, output string) {
	cmd.Command = dockerCli.Command("app", "image", "ls")
	result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
	assert.Equal(t, result.Stdout(), output)
}

func TestImageList(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		dir := fs.NewDir(t, "")
		defer dir.Remove()

		insertBundles(t, cmd, info)

		expected := `APP IMAGE                     APP NAME
%s push-pull
a-simple-app:latest           simple
b-simple-app:latest           simple
`
		expectedOutput := fmt.Sprintf(expected, info.registryAddress+"/c-myapp:latest")
		expectImageListOutput(t, cmd, expectedOutput)
	})
}

func TestImageRm(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		dir := fs.NewDir(t, "")
		defer dir.Remove()

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
			Err:      `Error: no such image b-simple-app:latest`,
		})

		expectedOutput := "APP IMAGE APP NAME\n"
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
		cmd.Command = dockerCli.Command("app", "build", filepath.Join("testdata", "simple", "simple.dockerapp"), "a-simple-app")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		singleImageExpectation := `APP IMAGE           APP NAME
a-simple-app:latest simple
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
			Err:      `could not parse 'a-simple-app$2' as a valid reference: invalid reference format`,
		})

		// with invalid target reference
		dockerAppImageTag("a-simple-app", "b@simple-app")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `could not parse 'b@simple-app' as a valid reference: invalid reference format`,
		})

		// with unexisting source image
		dockerAppImageTag("b-simple-app", "target")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `could not tag 'b-simple-app': no such application image`,
		})

		// with unexisting source tag
		dockerAppImageTag("a-simple-app:not-a-tag", "target")
		icmd.RunCmd(cmd).Assert(t, icmd.Expected{
			ExitCode: 1,
			Err:      `could not tag 'a-simple-app:not-a-tag': no such application image`,
		})

		// tag image with only names
		dockerAppImageTag("a-simple-app", "b-simple-app")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `APP IMAGE           APP NAME
a-simple-app:latest simple
b-simple-app:latest simple
`)

		// target tag
		dockerAppImageTag("a-simple-app", "a-simple-app:0.1")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `APP IMAGE           APP NAME
a-simple-app:0.1    simple
a-simple-app:latest simple
b-simple-app:latest simple
`)

		// source tag
		dockerAppImageTag("a-simple-app:0.1", "c-simple-app")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `APP IMAGE           APP NAME
a-simple-app:0.1    simple
a-simple-app:latest simple
b-simple-app:latest simple
c-simple-app:latest simple
`)

		// source and target tags
		dockerAppImageTag("a-simple-app:0.1", "b-simple-app:0.2")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `APP IMAGE           APP NAME
a-simple-app:0.1    simple
a-simple-app:latest simple
b-simple-app:0.2    simple
b-simple-app:latest simple
c-simple-app:latest simple
`)

		// given a new application
		cmd.Command = dockerCli.Command("app", "build", filepath.Join("testdata", "push-pull", "push-pull.dockerapp"), "push-pull")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `APP IMAGE           APP NAME
a-simple-app:0.1    simple
a-simple-app:latest simple
b-simple-app:0.2    simple
b-simple-app:latest simple
c-simple-app:latest simple
push-pull:latest    push-pull
`)

		// can be tagged to an existing tag
		dockerAppImageTag("push-pull", "b-simple-app:0.2")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		expectImageListOutput(t, cmd, `APP IMAGE           APP NAME
a-simple-app:0.1    simple
a-simple-app:latest simple
b-simple-app:0.2    push-pull
b-simple-app:latest simple
c-simple-app:latest simple
push-pull:latest    push-pull
`)
	})
}
