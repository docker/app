package main

import (
	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

var (
	forkMaintainers []string
	outputDir       string
)

func forkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fork <origin-name> <fork-name> [-p outputdir] [-m name:email ...]",
		Short: "Create a fork of an existing application to be modified",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return packager.Fork(args[0], args[1], outputDir, forkMaintainers)
		},
	}
	cmd.Flags().StringArrayVarP(&forkMaintainers, "maintainer", "m", []string{}, "Maintainer (name:email) (optional)")
	cmd.Flags().StringVarP(&outputDir, "path", "p", ".", "Directory where the application will be extracted")

	return cmd
}
