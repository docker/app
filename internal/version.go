package internal

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	// Version is the git tag that this was built from.
	Version = "unknown"
	// GitCommit is the commit that this was built from.
	GitCommit = "unknown"
)

// FullVersion returns a string of version information.
func FullVersion() string {
	res := []string{
		fmt.Sprintf("Version:               %s", Version),
		fmt.Sprintf("Git commit:            %s", GitCommit),
		fmt.Sprintf("OS/Arch:               %s/%s", runtime.GOOS, runtime.GOARCH),
	}

	return strings.Join(res, "\n")
}
