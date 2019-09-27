package e2e

import (
	"fmt"
	"path/filepath"
	"regexp"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/icmd"
)

var (
	reg      = regexp.MustCompile("Digest is (.*).")
	expected = `REPOSITORY             TAG    APP NAME
%s        push-pull
a-simple-app           latest simple
b-simple-app           latest simple
`
)

func TestImageList(t *testing.T) {
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		dir := fs.NewDir(t, "")
		defer dir.Remove()

		// Push an application so that we can later pull it by digest
		cmd.Command = dockerCli.Command("app", "push", "--tag", info.registryAddress+"/c-myapp", "--insecure-registries="+info.registryAddress, filepath.Join("testdata", "push-pull", "push-pull.dockerapp"))
		r := icmd.RunCmd(cmd).Assert(t, icmd.Success)

		// Get the digest from the output of the pull command
		out := r.Stdout()
		matches := reg.FindAllStringSubmatch(out, 1)
		digest := matches[0][1]

		// Pull the app by digest
		cmd.Command = dockerCli.Command("app", "pull", "--insecure-registries="+info.registryAddress, info.registryAddress+"/c-myapp@"+digest)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "bundle", filepath.Join("testdata", "simple", "simple.dockerapp"), "--tag", "b-simple-app", "--output", dir.Join("simple-bundle.json"))
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
		cmd.Command = dockerCli.Command("app", "bundle", filepath.Join("testdata", "simple", "simple.dockerapp"), "--tag", "a-simple-app", "--output", dir.Join("simple-bundle.json"))
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		expectedOutput := fmt.Sprintf(expected, info.registryAddress+"/c-myapp")
		cmd.Command = dockerCli.Command("app", "image", "ls")
		result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
		assert.Equal(t, result.Stdout(), expectedOutput)
	})
}
