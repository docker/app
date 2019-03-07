package e2e

import (
	"regexp"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/golden"
	"gotest.tools/icmd"
)

func TestInvokePluginFromCLI(t *testing.T) {
	// docker --help should list app as a top command
	cmd := icmd.Cmd{Command: dockerCli.Command("--help")}
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{
		Out: "app*        Docker Application Packages (Docker Inc.,",
	})

	// docker app --help prints docker-app help
	cmd.Command = dockerCli.Command("app", "--help")
	usage := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()

	goldenFile := "plugin-usage.golden"
	if hasExperimental {
		goldenFile = "plugin-usage-experimental.golden"
	}
	golden.Assert(t, usage, goldenFile)

	// docker info should print app version and short description
	cmd.Command = dockerCli.Command("info")
	re := regexp.MustCompile(`app: Docker Application Packages \(Docker Inc\., .*\)`)
	output := icmd.RunCmd(cmd).Assert(t, icmd.Success).Combined()
	assert.Assert(t, re.MatchString(output))
}
