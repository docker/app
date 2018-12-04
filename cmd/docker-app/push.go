package main

import (
	"fmt"

	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	namespace string
	tag       string
	repo      string
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
			err = runValidation(app)
			if err != nil {
				return err
			}
			dgst, err := packager.Push(app, opts.namespace, opts.tag, opts.repo)
			if err == nil {
				fmt.Println(dgst)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&opts.namespace, "namespace", "", "Namespace to use (default: namespace in metadata)")
	cmd.Flags().StringVarP(&opts.tag, "tag", "t", "", "Tag to use (default: version in metadata)")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "Name of the remote repository (default: <app-name>.dockerapp)")
	return cmd
}
