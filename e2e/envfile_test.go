package e2e

import (
	"path/filepath"
	"testing"

	"gotest.tools/icmd"
)

func TestRenderWithEnvFile(t *testing.T) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()
	appPath := filepath.Join("testdata", "envfile", "envfile.dockerapp")

	cmd.Command = dockerCli.Command("app", "build", "-f", appPath, "--tag", "a-simple-tag", "--no-resolve-image", ".")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Command = dockerCli.Command("app", "image", "render", "a-simple-tag")
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{Out: `version: "3.7"
services:
  db:
    environment:
      COMPANY: mycompany
      SOME_FILE: /some/file
      USER: myuser
    image: busybox:1.30.1`})
}
