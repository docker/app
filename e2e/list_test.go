package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/icmd"
)

func lineByLineComparator(t *testing.T, actual string, length int, expectedLines map[int]func(string) error) {
	t.Helper()
	lines := strings.Split(actual, "\n")
	assert.Equal(t, len(lines), length)
	if len(expectedLines) == 0 {
		return
	}
	for i, line := range lines {
		cmp, ok := expectedLines[i]
		if !ok {
			continue
		}
		if err := cmp(line); err != nil {
			t.Errorf("line %d: %s", i, err)
		}
	}
	if t.Failed() {
		t.Log(actual)
	}
}

func prefix(expected string) func(string) error {
	return func(actual string) error {
		if strings.HasPrefix(actual, expected) {
			return nil
		}
		return errors.Errorf("expected %q to start with %q", actual, expected)
	}
}

func equals(expected string) func(string) error {
	return func(actual string) error {
		if expected == actual {
			return nil
		}
		return errors.Errorf("got %q, expected %q", actual, expected)
	}
}

func extractImageID(t *testing.T, line string) string {
	fields := strings.Fields(line)
	assert.Assert(t, len(fields) > 3)
	return fields[2]
}

type commandConfig struct {
	env string
	exe string
	dir string
	t   *testing.T
}

func (c *commandConfig) run(args ...string) *icmd.Result {
	c.t.Helper()
	cmd := icmd.Command(c.exe, args...)
	cmd.Env = append(os.Environ(), c.env)
	cmd.Dir = c.dir
	result := icmd.RunCmd(cmd)
	result.Assert(c.t, icmd.Success)
	return result
}

func TestLsCmd(t *testing.T) {
	app, _ := getBinary(t)
	dind := startDind(t)
	defer dind.Stop(t)
	dockerApp := &commandConfig{
		env: fmt.Sprintf("DOCKER_HOST=%v", dind.Address(t)),
		exe: app,
		t:   t,
		dir: fs.NewDir(t, "test_docker_app_ls_cmd").Path(),
	}
	result := dockerApp.run("ls")
	lineByLineComparator(t, result.Stdout(), 2, map[int]func(string) error{
		0: equals("REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE"),
		1: equals(""),
	})
	// Create two apps.
	dockerApp.run("init", "ls_myapp")
	dockerApp.run("save", "ls_myapp.dockerapp")
	// We need to sleep between creation to avoid timestamp collision.
	time.Sleep(1 * time.Second)
	dockerApp.run("init", "anotherapp")
	dockerApp.run("save", "anotherapp.dockerapp")
	// We need to sleep again to avoid: CREATED `Less than a second ago`, otherwise, the header size is not constant.
	time.Sleep(1 * time.Second)

	// Except the output to contain both applications.
	result = dockerApp.run("ls")
	lineByLineComparator(t, result.Stdout(), 4, map[int]func(string) error{
		0: equals("REPOSITORY             TAG                 IMAGE ID            CREATED             SIZE"),
		1: prefix("anotherapp.dockerapp   0.1.0"),
		2: prefix("ls_myapp.dockerapp     0.1.0"),
		3: equals(""),
	})

	// Except quiet flag to return IDs only.
	var ids []string
	for _, line := range strings.Split(result.Stdout(), "\n")[1:2] {
		ids = append(ids, extractImageID(t, line))
	}
	result = dockerApp.run("ls", "--quiet")
	assert.DeepEqual(t, strings.Split(result.Stdout(), "\n")[0:1], ids)

	// Except the output to contain only ls_myapp.
	result = dockerApp.run("ls", "ls_myapp.dockerapp")
	lineByLineComparator(t, result.Stdout(), 3, map[int]func(string) error{
		0: equals("REPOSITORY           TAG                 IMAGE ID            CREATED             SIZE"),
		1: prefix("ls_myapp.dockerapp   0.1.0"),
		2: equals(""),
	})

	// Except quiet flag to return only one ID.
	id := extractImageID(t, strings.Split(result.Stdout(), "\n")[1])
	result = dockerApp.run("ls", "-q", "ls_myapp.dockerapp")
	assert.Equal(t, result.Stdout(), id+"\n")
}
