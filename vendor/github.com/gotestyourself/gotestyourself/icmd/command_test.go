package icmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/gotestyourself/gotestyourself/assert"
	"github.com/gotestyourself/gotestyourself/fs"
	"github.com/gotestyourself/gotestyourself/internal/maint"
)

var (
	bindir   = fs.NewDir(maint.T, "icmd-dir")
	binname  = bindir.Join("bin-stub") + pathext()
	stubpath = filepath.FromSlash("./internal/stub")
)

func pathext() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func TestMain(m *testing.M) {
	exitcode := m.Run()
	bindir.Remove()
	os.Exit(exitcode)
}

func buildStub(t assert.TestingT) {
	if _, err := os.Stat(binname); err == nil {
		return
	}
	result := RunCommand("go", "build", "-o", binname, stubpath)
	result.Assert(t, Success)
}

func TestRunCommandSuccess(t *testing.T) {
	buildStub(t)

	result := RunCommand(binname)
	result.Assert(t, Success)
}

func TestRunCommandWithCombined(t *testing.T) {
	buildStub(t)

	result := RunCommand(binname, "-warn")
	result.Assert(t, Expected{})

	assert.Equal(t, result.Combined(), "this is stdout\nthis is stderr\n")
	assert.Equal(t, result.Stdout(), "this is stdout\n")
	assert.Equal(t, result.Stderr(), "this is stderr\n")
}

func TestRunCommandWithTimeoutFinished(t *testing.T) {
	buildStub(t)

	result := RunCmd(Cmd{
		Command: []string{binname, "-sleep=1ms"},
		Timeout: 2 * time.Second,
	})
	result.Assert(t, Expected{Out: "this is stdout"})
}

func TestRunCommandWithTimeoutKilled(t *testing.T) {
	buildStub(t)

	command := []string{binname, "-sleep=200ms"}
	result := RunCmd(Cmd{Command: command, Timeout: 30 * time.Millisecond})
	result.Assert(t, Expected{Timeout: true, Out: None, Err: None})
}

func TestRunCommandWithErrors(t *testing.T) {
	buildStub(t)

	result := RunCommand("doesnotexists")
	expected := `exec: "doesnotexists": executable file not found`
	result.Assert(t, Expected{Out: None, Err: None, ExitCode: 127, Error: expected})
}

func TestRunCommandWithStdoutNoStderr(t *testing.T) {
	buildStub(t)

	result := RunCommand(binname)
	result.Assert(t, Expected{Out: "this is stdout\n", Err: None})
}

func TestRunCommandWithExitCode(t *testing.T) {
	buildStub(t)

	result := RunCommand(binname, "-fail=99")
	result.Assert(t, Expected{
		ExitCode: 99,
		Error:    "exit status 99",
	})
}
