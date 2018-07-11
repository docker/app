package main

import (
	"github.com/docker/app/internal/packager"
	"github.com/spf13/cobra"
)

var unpackOutputDir string

func unpackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unpack <app-name> [-o output_dir]",
		Short: "Unpack the application to expose the content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return packager.Unpack(firstOrEmpty(args), unpackOutputDir)
		},
	}
	cmd.Flags().StringVarP(&unpackOutputDir, "output", "o", ".", "Output directory (.)")
	return cmd
}
