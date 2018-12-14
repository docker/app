package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/deislabs/duffle/pkg/duffle/home"
	"github.com/deislabs/duffle/pkg/duffle/manifest"
)

const createDesc = `
This command creates a bundle directory along with a minimal duffle.toml file and a cnab directory with a Dockerfile for the invocation image

For example, 'duffle create foo'  will create a directory structure that looks like this:

    foo/
        |
        |- duffle.yaml        # Contains metadata for bundle
        |
        |- cnab/              # Contains invocation image
                |
                |- Dockerfile     # Dockerfile for invocation image

If directories in the given path do not exist, it will attempt to create them. If the given path exists and there are files in that directory, conflicting files will be overwritten but other files will be left alone.
`

type createCmd struct {
	path string
	home home.Home
	out  io.Writer
}

func newCreateCmd(w io.Writer) *cobra.Command {
	create := &createCmd{out: w}

	cmd := &cobra.Command{
		Use:   "create [PATH]",
		Short: "scaffold a bundle",
		Long:  createDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("this command requires the path")
			}
			create.home = home.Home(homePath())
			create.path = args[0]

			return create.run()
		},
	}

	return cmd
}

func (c *createCmd) run() error {
	path, err := filepath.Abs(c.path)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.out, "Creating %s\n", c.path)
	if err := os.Mkdir(path, 0755); err != nil {
		return err
	}

	return manifest.Scaffold(path)
}
