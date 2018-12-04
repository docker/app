package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/deis/duffle/pkg/credentials"
	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/docker/app/internal/yaml"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

type credentialSetOptions struct {
	name        string
	contextName string
	force       bool
	output      string
}

func credentialSetCmd(dockerCli command.Cli) *cobra.Command {
	var opts credentialSetOptions
	cmd := &cobra.Command{
		Use:   "add-credentialset <name> <docker context>",
		Short: "Add a CNAB credentialset in the credential store for the given Docker Context",
		Args:  cli.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			opts.contextName = args[1]
			return runCredentials(dockerCli, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.force, "force", false, "Overwrites existing credentialset")
	cmd.Flags().StringVar(&opts.output, "out", "", "Specify an alternate output for the credentialset (- for stdout)")
	return cmd
}

func runCredentials(dockerCli command.Cli, opts credentialSetOptions) error {
	output := opts.output
	var writer io.Writer
	if output == "-" {
		writer = dockerCli.Out()
	} else {
		if output == "" {
			h := home.Home(home.DefaultHome())
			output = filepath.Join(h.Credentials(), opts.name) + ".yaml"
		}
		if _, err := os.Stat(output); err == nil && !opts.force {
			return fmt.Errorf("credentialset %q already exists, use --force to overwrite", opts.name)
		}
		f, err := os.OpenFile(output, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		writer = f
	}
	creds := credentials.CredentialSet{
		Name: opts.name,
		Credentials: []credentials.CredentialStrategy{
			{
				Name: "docker.context",
				Source: credentials.Source{
					Command: fmt.Sprintf(`docker-app context export %s -`, opts.contextName),
				},
			},
		},
	}
	payload, err := yaml.Marshal(creds)
	if err != nil {
		return err
	}
	_, err = writer.Write(payload)
	return err
}
