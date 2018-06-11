package main

import (
	"fmt"

	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

type saveOptions struct {
	namespace string
	tag       string
}

func saveCmd() *cobra.Command {
	var opts saveOptions
	cmd := &cobra.Command{
		Use:   "save [<app-name>]",
		Short: "Save the application as an image to the docker daemon(in preparation for push)",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			imageName, err := packager.Save(firstOrEmpty(args), opts.namespace, opts.tag)
			if imageName != "" {
				fmt.Printf("Saved application as image: %s\n", imageName)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&opts.namespace, "namespace", "", "namespace to use (default: namespace in metadata)")
	cmd.Flags().StringVarP(&opts.tag, "tag", "t", "", "tag to use (default: version in metadata)")
	return cmd
}
