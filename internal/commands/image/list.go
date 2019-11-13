package image

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/docker/cli/templates"
	"github.com/pkg/errors"

	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/relocated"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/stringid"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
)

type imageListOption struct {
	quiet    bool
	digests  bool
	template string
}

type imageListColumn struct {
	header string
	value  func(desc imageDesc) string
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
	flags.BoolVarP(&options.digests, "digests", "", false, "Show image digests")
	cmd.Flags().StringVarP(&options.template, "format", "f", "", "Format the output using the given syntax or Go template")
	cmd.Flags().SetAnnotation("format", "experimentalCLI", []string{"true"}) //nolint:errcheck

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
	return printImages(dockerCli, pkgs, options)
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

func printImages(dockerCli command.Cli, refs []pkg, options imageListOption) error {
	w := tabwriter.NewWriter(dockerCli.Out(), 0, 0, 1, ' ', 0)

	list := []imageDesc{}
	for _, ref := range refs {
		list = append(list, getImageDesc(ref))
	}

	if options.template == "json" {
		bytes, err := json.MarshalIndent(list, "", "  ")
		if err != nil {
			return errors.Errorf("Failed to marshall json: %s", err)
		}
		_, err = dockerCli.Out().Write(bytes)
		return err
	}
	if options.template != "" {
		tmpl, err := templates.Parse(options.template)
		if err != nil {
			return errors.Errorf("Template parsing error: %s", err)
		}
		return tmpl.Execute(dockerCli.Out(), list)
	}

	listColumns := getImageListColumns(options)
	printHeaders(w, listColumns)
	for _, desc := range list {
		printValues(w, desc, listColumns)
	}

	return w.Flush()
}

func printImageIDs(dockerCli command.Cli, refs []pkg) error {
	var buf bytes.Buffer

	for _, ref := range refs {
		id, err := getImageID(ref)
		if err != nil {
			return err
		}
		fmt.Fprintln(&buf, id)
	}
	fmt.Fprint(dockerCli.Out(), buf.String())
	return nil
}

func getImageID(p pkg) (string, error) {
	id, ok := p.ref.(store.ID)
	if !ok {
		var err error
		id, err = store.FromBundle(p.bundle)
		if err != nil {
			return "", err
		}
	}
	return stringid.TruncateID(id.String()), nil
}

func printHeaders(w io.Writer, listColumns []imageListColumn) {
	var headers []string
	for _, column := range listColumns {
		headers = append(headers, column.header)
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))
}

func printValues(w io.Writer, desc imageDesc, listColumns []imageListColumn) {
	var values []string
	for _, column := range listColumns {
		values = append(values, column.value(desc))
	}
	fmt.Fprintln(w, strings.Join(values, "\t"))
}

type imageDesc struct {
	ID         string        `json:"id,omitempty"`
	Name       string        `json:"name,omitempty"`
	Repository string        `json:"repository,omitempty"`
	Tag        string        `json:"tag,omitempty"`
	Digest     string        `json:"digest,omitempty"`
	Created    time.Duration `json:"created,omitempty"`
}

func getImageDesc(p pkg) imageDesc {
	var id string
	id, _ = getImageID(p)
	var repository string
	if n, ok := p.ref.(reference.Named); ok {
		repository = reference.FamiliarName(n)
	}
	var tag string
	if t, ok := p.ref.(reference.Tagged); ok {
		tag = t.Tag()
	}
	var digest string
	if t, ok := p.ref.(reference.Digested); ok {
		digest = t.Digest().String()
	}
	var created time.Duration
	if payload, err := packager.CustomPayload(p.bundle.Bundle); err == nil {
		if createdPayload, ok := payload.(packager.CustomPayloadCreated); ok {
			created = time.Now().UTC().Sub(createdPayload.CreatedTime())
		}
	}
	return imageDesc{
		ID:         id,
		Name:       p.bundle.Name,
		Repository: repository,
		Tag:        tag,
		Digest:     digest,
		Created:    created,
	}
}

func getImageListColumns(options imageListOption) []imageListColumn {
	columns := []imageListColumn{
		{"REPOSITORY", func(desc imageDesc) string {
			if desc.Repository != "" {
				return desc.Repository
			}
			return "<none>"
		}},
		{"TAG", func(desc imageDesc) string {
			if desc.Tag != "" {
				return desc.Tag
			}
			return "<none>"
		}},
	}
	if options.digests {
		columns = append(columns, imageListColumn{"DIGEST", func(desc imageDesc) string {
			if desc.Digest != "" {
				return desc.Digest
			}
			return "<none>"
		}})
	}
	columns = append(columns,
		imageListColumn{"APP IMAGE ID", func(desc imageDesc) string {
			return desc.ID
		}},
		imageListColumn{"APP NAME", func(desc imageDesc) string {
			return desc.Name
		}},
		imageListColumn{"CREATED", func(desc imageDesc) string {
			if desc.Created > 0 {
				return units.HumanDuration(desc.Created) + " ago"
			}
			return ""
		}},
	)
	return columns
}

type pkg struct {
	ref    reference.Reference
	bundle *relocated.Bundle
}
