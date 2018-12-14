package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/deislabs/duffle/pkg/duffle/home"
	"github.com/deislabs/duffle/pkg/packager"

	"github.com/spf13/cobra"
)

const exportDesc = `
Packages a bundle, invocation images, and all referenced images within a single
gzipped tarfile.

All images specified in the bundle metadata are saved as tar files in the artifacts/
directory along with an artifacts.json file which describes the contents of artifacts/.

By default, this command will use the name and version information of the bundle to create
a compressed archive file called <name>-<version>.tgz in the current directory. This
behavior can be augmented by specifying a file path to save the compressed bundle to using
the --output-file flag.
`

type exportCmd struct {
	dest    string
	path    string
	home    home.Home
	out     io.Writer
	full    bool
	verbose bool
}

func newExportCmd(w io.Writer) *cobra.Command {
	export := &exportCmd{out: w}

	cmd := &cobra.Command{
		Use:   "export [PATH]",
		Short: "package CNAB bundle in gzipped tar file",
		Long:  exportDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("this command requires the path to the bundle")
			}
			export.home = home.Home(homePath())
			export.path = args[0]

			return export.run()
		},
	}

	f := cmd.Flags()
	f.StringVarP(&export.dest, "output-file", "o", "", "Save exported bundle to file path")
	f.BoolVarP(&export.full, "full", "u", true, "Save bundle with all associated images")
	f.BoolVarP(&export.verbose, "verbose", "v", false, "Verbose output")

	return cmd
}

func (ex *exportCmd) run() error {
	source, err := filepath.Abs(ex.path)
	if err != nil {
		return err
	}

	exp, err := packager.NewExporter(source, ex.dest, ex.home.Logs(), ex.full)
	if err != nil {
		return fmt.Errorf("Unable to set up exporter: %s", err)
	}
	if err := exp.Export(); err != nil {
		return err
	}
	if ex.verbose {
		fmt.Fprintf(ex.out, "Export logs: %s\n", exp.Logs)
	}
	return nil
}
