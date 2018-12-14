package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
	"github.com/spf13/cobra"
)

type pullOptions struct {
	insecure bool
	out      string
}

func pullCmd(dockerCli command.Cli) *cobra.Command {
	var opts pullOptions
	cmd := &cobra.Command{
		Use:   "pull <repotag>",
		Short: "Pull an application from a registry",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := pullBundle(dockerCli, args[0], true, opts.insecure)
			if err != nil {
				return err
			}
			bundleJSON, err := json.MarshalIndent(b, "", "  ")
			if err != nil {
				return err
			}
			if opts.out == "-" {
				fmt.Print(bundleJSON)
				return nil
			}
			return ioutil.WriteFile(opts.out, bundleJSON, 0644)
		},
	}
	cmd.Flags().BoolVar(&opts.insecure, "insecure", false, "Use insecure registry, without SSL")
	cmd.Flags().StringVarP(&opts.out, "out", "o", "bundle.json", "path to the output bundle.json (- for stdout)")
	return cmd
}

func pullBundle(dockerCli command.Cli, name string, force, insecure bool) (*bundle.Bundle, error) {
	named, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return nil, err
	}
	resolver := remotes.CreateResolver(dockerCli.ConfigFile(), insecure)
	return remotes.Pull(context.Background(), named, resolver)
}
