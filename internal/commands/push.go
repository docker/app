package commands

import (
	"fmt"

	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	tag string
}

func pushCmd() *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:   "push [<app-name>]",
		Short: "Push the application to a registry",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := packager.Extract(firstOrEmpty(args))
			if err != nil {
				return err
			}
			defer app.Cleanup()
			dgst, err := packager.Push(app, opts.tag)
			if err == nil {
				fmt.Println(dgst)
			}
			return err
		},
	}
	cmd.Flags().StringVarP(&opts.tag, "tag", "t", "", "Target registry reference (default is : from metadata)")
	return cmd
}
