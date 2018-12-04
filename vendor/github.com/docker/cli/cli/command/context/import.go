package context

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/store"
	"github.com/spf13/cobra"
)

type importOptions struct {
	force bool
	name  string
}

func newImportCommand(dockerCli command.Cli) *cobra.Command {
	opts := &importOptions{}
	cmd := &cobra.Command{
		Use:   "import <filename> [OPTIONS]",
		Short: "Import a context",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.name == "" {
				return errors.New("name is required")
			}
			file := args[0]
			_, err := dockerCli.ContextStore().GetContextMetadata(opts.name)
			exists := err == nil
			if exists && !opts.force {
				return fmt.Errorf("context %q already exists", opts.name)
			}
			var reader io.Reader
			if file == "-" {
				reader = dockerCli.In()
			} else {
				f, err := os.Open(file)
				if err != nil {
					return err
				}
				defer f.Close()
				reader = f
			}

			if err := store.Import(opts.name, dockerCli.ContextStore(), reader); err != nil {
				return err
			}

			return nil
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&opts.force, "force", false, "overwrite any existing context with the same name")
	flags.StringVar(&opts.name, "name", "", "name of the context")
	return cmd
}
