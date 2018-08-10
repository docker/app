package main

import (
	"os"

	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var splitOutputDir string

func splitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "split [<app-name>] [-o output]",
		Short: "Split a single-file application into multiple files",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			extractedApp, err := packager.Extract(firstOrEmpty(args))
			if err != nil {
				return err
			}
			defer extractedApp.Cleanup()
			inPlace := splitOutputDir == ""
			if inPlace {
				splitOutputDir = extractedApp.Path + ".tmp"
			}
			if err := packager.Split(extractedApp, splitOutputDir); err != nil {
				return err
			}
			if inPlace {
				if err := os.RemoveAll(extractedApp.Path); err != nil {
					return errors.Wrap(err, "failed to erase previous application directory")
				}
				if err := os.Rename(splitOutputDir, extractedApp.Path); err != nil {
					return errors.Wrap(err, "failed to rename new application directory")
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&splitOutputDir, "output", "o", "", "Output application directory (default: in-place)")
	return cmd
}
