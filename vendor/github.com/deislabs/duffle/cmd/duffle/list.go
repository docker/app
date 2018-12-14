package main

import (
	"fmt"
	"io"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"
)

type listCmd struct {
	out  io.Writer
	long bool
}

func newListCmd(out io.Writer) *cobra.Command {
	list := listCmd{out: out}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list installed apps",
		RunE: func(cmd *cobra.Command, args []string) error {
			return list.run()
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&list.long, "long", "l", false, "output longer listing format")

	return cmd
}

func (l *listCmd) run() error {
	if !l.long {
		claims, err := claimStorage().List()
		if err != nil {
			return err
		}
		for _, claim := range claims {
			fmt.Fprintln(l.out, claim)
		}

	} else {
		claims, err := claimStorage().ReadAll()
		if err != nil {
			return err
		}

		table := uitable.New()
		table.MaxColWidth = 50
		table.Wrap = true

		table.AddRow("NAME", "BUNDLE", "INSTALLED", "LAST ACTION", "LAST STATUS")
		for _, cl := range claims {
			table.AddRow(cl.Name, cl.Bundle.Name, cl.Created, cl.Result.Action, cl.Result.Status)
		}

		fmt.Fprintln(l.out, table)
	}

	return nil
}
