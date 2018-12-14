package main

import (
	"fmt"
	"io"

	"github.com/deislabs/duffle/pkg/version"

	"github.com/spf13/cobra"
)

func newVersionCmd(w io.Writer) *cobra.Command {
	const usage = `print current version of the Duffle CLI`

	cmd := &cobra.Command{
		Use:   "version",
		Short: usage,
		Long:  usage,
		Run: func(cmd *cobra.Command, args []string) {
			showVersion(cmd.OutOrStdout())
		},
	}

	return cmd
}

func showVersion(out io.Writer) {
	fmt.Fprintln(out, version.Version)
}
