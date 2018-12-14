package main

import (
	"io"

	"github.com/spf13/cobra"
)

func newInspectCmd(w io.Writer) *cobra.Command {
	var (
		insecure bool
	)

	const usage = ` Returns information about an application bundle.

	Example:
		$ duffle inspect duffle/example:0.1.0

	To inspect unsigned bundles, pass the --insecure flag:
		$ duffle inspect duffle/unsinged-example:0.1.0 --insecure
`

	cmd := &cobra.Command{
		Use:   "inspect NAME",
		Short: "return low-level information on application bundles",
		Long:  usage,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bundleName := args[0]

			bundleFile, err := getBundleFilepath(bundleName, insecure)
			if err != nil {
				return err
			}

			bun, err := loadBundle(bundleFile, insecure)
			if err != nil {
				return err
			}

			_, err = bun.WriteTo(w)

			return err
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&insecure, "insecure", "k", false, "Do not verify the bundle (INSECURE)")

	return cmd
}
