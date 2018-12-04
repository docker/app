package main

import (
	"errors"
	"io"
	"path/filepath"

	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/packager"

	"github.com/spf13/cobra"
)

const importDesc = `
Unpacks a bundle from a gzipped tar file on local file system
`

type importCmd struct {
	source   string
	dest     string
	out      io.Writer
	home     home.Home
	insecure bool
	verbose  bool
}

func newImportCmd(w io.Writer) *cobra.Command {
	importc := &importCmd{
		out:  w,
		home: home.Home(homePath()),
	}

	cmd := &cobra.Command{
		Use:   "import [PATH]",
		Short: "unpack CNAB bundle from gzipped tar file",
		Long:  importDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("this command requires the path to the packaged bundle")
			}
			importc.source = args[0]

			return importc.run()
		},
	}

	f := cmd.Flags()
	f.StringVarP(&importc.dest, "destination", "d", "", "Location to unpack bundle")
	f.BoolVarP(&importc.insecure, "insecure", "k", false, "Do not verify the bundle (INSECURE)")
	f.BoolVarP(&importc.verbose, "verbose", "v", false, "Verbose output")

	return cmd
}

func (im *importCmd) run() error {
	source, err := filepath.Abs(im.source)
	if err != nil {
		return err
	}

	dest, err := filepath.Abs(im.dest) //TODO: double check
	if err != nil {
		return err
	}

	l, err := getLoader(im.insecure)
	if err != nil {
		return err
	}

	imp, err := packager.NewImporter(source, dest, l, im.verbose)
	if err != nil {
		return err
	}
	return imp.Import()
}
