package main

import (
	"github.com/spf13/cobra"
)

func imageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Manipulate Docker images associated with the application",
	}
	cmd.AddCommand(imageAddCmd(), imageLoadCmd(), imagePushCmd())
	return cmd
}
