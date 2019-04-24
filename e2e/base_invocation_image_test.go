package e2e

import (
	"fmt"
	"strings"
	"testing"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"

	dockerConfigFile "github.com/docker/cli/cli/config/configfile"
	"gotest.tools/assert"
	"gotest.tools/icmd"
)

func TestBaseInvocationImageVersion(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		cmd, cleanup := dockerCli.createTestCmd()
		defer cleanup()
		cmd.Command = dockerCli.Command("app", "version", "--base-invocation-image")
		result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
		output := strings.TrimSpace(result.Stdout())
		assert.Equal(t, output, fmt.Sprintf("%s:%s", packager.DefaultCNABBaseImageName, internal.Version))
	})

	t.Run("config", func(t *testing.T) {
		imageName := "some-base-image:some-tag"
		cmd, cleanup := dockerCli.createTestCmd(func(config *dockerConfigFile.ConfigFile) {
			config.SetPluginConfig("app", "base-invocation-image", imageName)
		})
		defer cleanup()
		cmd.Command = dockerCli.Command("app", "version", "--base-invocation-image")
		result := icmd.RunCmd(cmd).Assert(t, icmd.Success)
		output := strings.TrimSpace(result.Stdout())
		assert.Equal(t, output, imageName)
	})
}
