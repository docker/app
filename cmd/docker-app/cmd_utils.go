package main

import (
	cliconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/debug"
	cliflags "github.com/docker/cli/cli/flags"
)

func firstOrEmpty(list []string) string {
	if len(list) != 0 {
		return list[0]
	}
	return ""
}

func dockerPreRun(opts *cliflags.ClientOptions) {
	cliflags.SetLogLevel(opts.Common.LogLevel)
	if opts.ConfigDir != "" {
		cliconfig.SetDir(opts.ConfigDir)
	}
	if opts.Common.Debug {
		debug.Enable()
	}
}
