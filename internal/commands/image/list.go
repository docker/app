package image

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	"github.com/spf13/cobra"
)

type imageListOptions struct {
	showDigests bool
}

func listCmd(dockerCli command.Cli) *cobra.Command {
	options := imageListOptions{}
	cmd := &cobra.Command{
		Short:   "List application images",
		Use:     "ls",
		Aliases: []string{"list"},
		RunE: func(cmd *cobra.Command, args []string) error {
			appstore, err := store.NewApplicationStore(config.Dir())
			if err != nil {
				return err
			}

			bundleStore, err := appstore.BundleStore()
			if err != nil {
				return err
			}

			return runList(dockerCli, options, bundleStore)
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&options.showDigests, "digests", false, "Show digests")

	return cmd
}

func runList(dockerCli command.Cli, options imageListOptions, bundleStore store.BundleStore) error {
	bundles, err := bundleStore.List()
	if err != nil {
		return err
	}

	pkgs, err := getPackages(bundleStore, bundles)
	if err != nil {
		return err
	}

	return printImages(dockerCli, options, pkgs)
}

func getPackages(bundleStore store.BundleStore, references []reference.Named) ([]pkg, error) {
	packages := make([]pkg, len(references))
	for i, ref := range references {
		b, err := bundleStore.Read(ref)
		if err != nil {
			return nil, err
		}

		pk := pkg{
			bundle: b,
			ref:    ref,
		}

		packages[i] = pk
	}

	return packages, nil
}

func printImages(dockerCli command.Cli, options imageListOptions, refs []pkg) error {
	w := tabwriter.NewWriter(dockerCli.Out(), 0, 0, 1, ' ', 0)

	columns := getColumns(options)
	printHeaders(w, columns)
	for _, ref := range refs {
		printValues(w, columns, ref)
	}

	return w.Flush()
}

func getColumns(options imageListOptions) columns {
	if options.showDigests {
		return withDigestColumns
	}
	return defaultColumns
}

func printHeaders(w io.Writer, cols columns) {
	var headers []string
	for _, column := range cols {
		headers = append(headers, column.header)
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))
}

func printValues(w io.Writer, cols columns, ref pkg) {
	var values []string
	for _, column := range cols {
		values = append(values, column.value(ref))
	}
	fmt.Fprintln(w, strings.Join(values, "\t"))
}

var (
	defaultColumns = []column{
		{"APP IMAGE", func(p pkg) string {
			return reference.FamiliarString(p.ref)
		}},
		{"APP NAME", func(p pkg) string {
			return p.bundle.Name
		}},
	}

	withDigestColumns = columns{
		{"REPOSITORY", func(p pkg) string {
			return reference.FamiliarName(p.ref)
		}},
		{"TAG", func(p pkg) string {
			t, ok := p.ref.(reference.Tagged)
			if ok {
				return t.Tag()
			}
			return "<none>"
		}},
		{"DIGEST", func(p pkg) string {
			t, ok := p.ref.(reference.Digested)
			if ok {
				return t.Digest().Encoded()
			}
			return "<none>"
		}},
		{"APP NAME", func(p pkg) string {
			return p.bundle.Name
		}},
	}
)

type columns []column

type column struct {
	header string
	value  func(p pkg) string
}

type pkg struct {
	ref    reference.Named
	bundle *bundle.Bundle
}
