package internal

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

var (
	// Version is the git tag that this was built from.
	Version = "unknown"
	// GitCommit is the commit that this was built from.
	GitCommit = "unknown"
	// BuildTime is the time at which the binary was built.
	BuildTime = "unknown"
)

// FullVersion returns a string of version information.
func FullVersion() string {
	res := []string{
		fmt.Sprintf("Version:      %s", Version),
		fmt.Sprintf("Git commit:   %s", GitCommit),
		fmt.Sprintf("Built:        %s", reformatDate(BuildTime)),
		fmt.Sprintf("OS/Arch:      %s/%s", runtime.GOOS, runtime.GOARCH),
	}
	return strings.Join(res, "\n")
}

// FIXME(chris-crone): use function in docker/cli/cli/command/system/version.go.
func reformatDate(buildTime string) string {
	t, errTime := time.Parse(time.RFC3339Nano, buildTime)
	if errTime == nil {
		return t.Format(time.ANSIC)
	}
	return buildTime
}
