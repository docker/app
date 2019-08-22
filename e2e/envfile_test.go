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

	cmd.Command = dockerCli.Command("app", "render", appPath)
	icmd.RunCmd(cmd).Assert(t, icmd.Expected{Out: `version: "3.7"
services:
  db:
    environment:
      COMPANY: mycompany
      SOME_FILE: /some/file
      USER: myuser
    env_file:
    - myvars.env
    image: busybox:1.30.1`})
}
