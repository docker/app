package e2e

import (
	"path/filepath"
	"testing"

	"gotest.tools/icmd"
)

func TestRunTwice(t *testing.T) {
	// Test that we are indeed generating random app names
	// We had a problem where the second run would fail with an error
	// "Installation "gallant_poitras" already exists, use 'docker app update' instead"
	runWithDindSwarmAndRegistry(t, func(info dindSwarmAndRegistryInfo) {
		cmd := info.configuredCmd
		contextPath := filepath.Join("testdata", "simple")

		cmd.Command = dockerCli.Command("app", "build", "--tag", "myapp", contextPath)
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "run", "myapp", "--set", "web_port=8080")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)

		cmd.Command = dockerCli.Command("app", "run", "myapp", "--set", "web_port=8081")
		icmd.RunCmd(cmd).Assert(t, icmd.Success)
	})
}
