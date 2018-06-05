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
	// BuildTime is the time at which the binary was built.
	BuildTime = "unknown"
)

// FullVersion returns a string of version information.
func FullVersion() string {
	res := []string{
		fmt.Sprintf("Version:      %s", Version),
		fmt.Sprintf("Git commit:   %s", GitCommit),
		fmt.Sprintf("Build time:   %s", BuildTime),
		fmt.Sprintf("OS/Arch:      %s/%s", runtime.GOOS, runtime.GOARCH),
		fmt.Sprintf("Experimental: %s", Experimental),
		fmt.Sprintf("Renderers:    %s", Renderers),
	}
	return strings.Join(res, "\n")
}
