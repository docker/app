package main

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
)

var packOutputFile string

func packCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack [<app-name>] [-o output_file]",
		Short: "Pack the application as a single file",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appname := firstOrEmpty(args)
			appname, cleanup, err := packager.Extract(appname, nil)
			if err != nil {
				return err
			}
			defer cleanup()
			var target io.Writer
			if packOutputFile == "-" {
				if terminal.IsTerminal(int(dockerCli.Out().FD())) {
					return fmt.Errorf("Refusing to output to a terminal, use a shell redirect or the '-o' option")
				}
			} else {
				target, err = os.Create(packOutputFile)
				if err != nil {
					return err
				}
			}
			return packager.Pack(appname, target)
		},
	}
	cmd.Flags().StringVarP(&packOutputFile, "output", "o", "-", "Output file (- for stdout)")
	return cmd
}
