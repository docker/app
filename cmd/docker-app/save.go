package main

import (
	"fmt"

	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type saveOptions struct {
	namespace string
	tag       string
}

func saveCmd(dockerCli command.Cli) *cobra.Command {
	var opts saveOptions
	cmd := &cobra.Command{
		Use:   "save [<app-name>]",
		Short: "Save the application as an image to the docker daemon(in preparation for push)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			imageName, err := packager.Save(firstOrEmpty(args), opts.namespace, opts.tag)
			if imageName != "" && err == nil {
				fmt.Fprintf(dockerCli.Out(), "Saved application as image: %s\n", imageName)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&opts.namespace, "namespace", "", "namespace to use (default: namespace in metadata)")
	cmd.Flags().StringVarP(&opts.tag, "tag", "t", "", "tag to use (default: version in metadata)")
	return cmd
}
