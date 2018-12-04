package main

import (
	"io"

	"github.com/spf13/cobra"
)

// This is a useful class to embed in duffle commands to get common handlers
// for wiring up a command to cobra.
type duffleCmd struct {
	Out io.Writer
}

// Prepare implements the cobra.PreRunE function signature and wires up a duffle
// command for use with cobra.
func (dc duffleCmd) Prepare() func(*cobra.Command, []string) error {
	return func(cc *cobra.Command, strings []string) error {
		dc.Out = cc.OutOrStdout()
		return nil
	}
}
