package main

import (
	"io"
	"os"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

var mergeOutputFile string

func mergeCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge [<app-name>] [-o output_dir]",
		Short: "Merge the application as a single file multi-document YAML",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			appname, cleanup, err := packager.Extract(firstOrEmpty(args))
			if err != nil {
				return err
			}
			defer cleanup()
			var target io.Writer
			if mergeOutputFile == "-" {
				target = dockerCli.Out()
			} else {
				target, err = os.Create(mergeOutputFile)
				if err != nil {
					return err
				}
				defer target.(io.WriteCloser).Close()
			}
			return packager.Merge(appname, target)
		},
	}
	if internal.Experimental == "on" {
		cmd.Flags().StringVarP(&mergeOutputFile, "output", "o", "-", "Output file (default: stdout)")
	}
	return cmd
}
