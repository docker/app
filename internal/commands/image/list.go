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

func listCmd(dockerCli command.Cli) *cobra.Command {
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

			return runList(dockerCli, bundleStore)
		},
	}

	return cmd
}

func runList(dockerCli command.Cli, bundleStore store.BundleStore) error {
	bundles, err := bundleStore.List()
	if err != nil {
		return err
	}

	pkgs, err := getPackages(bundleStore, bundles)
	if err != nil {
		return err
	}

	return printImages(dockerCli, pkgs)
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

		if r, ok := ref.(reference.NamedTagged); ok {
			pk.taggedRef = r
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
		{"REPOSITORY", func(p pkg) string {
			return reference.FamiliarName(p.ref)
		}},
		{"TAG", func(p pkg) string {
			if p.taggedRef != nil {
				return p.taggedRef.Tag()
			}
			return ""
		}},
		{"APP NAME", func(p pkg) string {
			return p.bundle.Name
		}},
	}
)

type pkg struct {
	ref       reference.Named
	taggedRef reference.NamedTagged
	bundle    *bundle.Bundle
}
