package image

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/stringid"
	"github.com/spf13/cobra"
)

type imageListOption struct {
	quiet bool
}

func listCmd(dockerCli command.Cli) *cobra.Command {
	options := imageListOption{}
	cmd := &cobra.Command{
		Short:   "List App images",
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
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only show numeric IDs")

	return cmd
}

func runList(dockerCli command.Cli, options imageListOption, bundleStore store.BundleStore) error {
	bundles, err := bundleStore.List()
	if err != nil {
		return err
	}

	pkgs, err := getPackages(bundleStore, bundles)
	if err != nil {
		return err
	}

	if options.quiet {
		return printImageIDs(dockerCli, pkgs)
	}
	return printImages(dockerCli, pkgs)
}

func getPackages(bundleStore store.BundleStore, references []reference.Reference) ([]pkg, error) {
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

func printImages(dockerCli command.Cli, refs []pkg) error {
	w := tabwriter.NewWriter(dockerCli.Out(), 0, 0, 1, ' ', 0)

	printHeaders(w)
	for _, ref := range refs {
		printValues(w, ref)
	}

	return w.Flush()
}

func printImageIDs(dockerCli command.Cli, refs []pkg) error {
	var buf bytes.Buffer

	for _, ref := range refs {
		id, ok := ref.ref.(store.ID)
		if !ok {
			var err error
			id, err = store.FromBundle(ref.bundle)
			if err != nil {
				return err
			}
		}
		fmt.Fprintln(&buf, stringid.TruncateID(id.String()))
	}
	fmt.Fprint(dockerCli.Out(), buf.String())
	return nil
}

func printHeaders(w io.Writer) {
	var headers []string
	for _, column := range listColumns {
		headers = append(headers, column.header)
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))
}

func printValues(w io.Writer, ref pkg) {
	var values []string
	for _, column := range listColumns {
		values = append(values, column.value(ref))
	}
	fmt.Fprintln(w, strings.Join(values, "\t"))
}

var (
	listColumns = []struct {
		header string
		value  func(p pkg) string
	}{
		{"APP IMAGE", func(p pkg) string {
			return reference.FamiliarString(p.ref)
		}},
		{"APP NAME", func(p pkg) string {
			return p.bundle.Name
		}},
	}
)

type pkg struct {
	ref    reference.Reference
	bundle *bundle.Bundle
}
