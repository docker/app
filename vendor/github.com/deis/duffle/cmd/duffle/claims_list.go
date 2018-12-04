package main

import (
	"io"

	"github.com/spf13/cobra"
)

func newClaimListCmd(out io.Writer) *cobra.Command {
	list := listCmd{out: out}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "list available claims",
		RunE: func(cmd *cobra.Command, args []string) error {
			l := &listCmd{out: out, long: list.long}
			return l.run()
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&list.long, "long", "l", false, "output longer listing format")

	return cmd
}
