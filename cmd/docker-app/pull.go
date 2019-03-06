package main

import (
	"fmt"

	bundlestore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func pullCmd(dockerCli command.Cli) *cobra.Command {
	var opts registryOptions
	cmd := &cobra.Command{
		Use:   "pull <repotag>",
		Short: "Pull an application from a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPull(dockerCli, args[0], opts)
		},
	}
	opts.addFlags(cmd.Flags())
	return cmd
}

func runPull(dockerCli command.Cli, name string, opts registryOptions) error {
	ref, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return errors.Wrap(err, name)
	}
	bndl, err := bundlestore.LookupOrPullBundle(dockerCli, reference.TagNameOnly(ref), true, opts.insecureRegistries)
	if err != nil {
		return errors.Wrap(err, name)
	}

	fmt.Printf("Successfully pulled %q (%s) from %s\n", bndl.Name, bndl.Version, ref.String())

	return nil
}
