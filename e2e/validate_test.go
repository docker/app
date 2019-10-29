package e2e

import (
	"path"
	"strings"
	"testing"

	"gotest.tools/assert"

	"gotest.tools/icmd"
)

func TestOrphanedParameter(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	p := path.Join("testdata", "invalid", "unused_parameter")
	cmd.Command = dockerCli.Command("app", "validate", p)
	out := icmd.RunCmd(cmd).Assert(t, icmd.Expected{ExitCode: 1}).Combined()
	assert.Assert(t, strings.Contains(out, "unused.parameter is declared as parameter but not used by the compose file"))
}
