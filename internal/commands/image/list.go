package image

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/relocated"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/templates"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/stringid"
	units "github.com/docker/go-units"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type imageListOption struct {
	quiet    bool
	digests  bool
	template string
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
	images, err := getImageDescriptors(bundleStore)
	if err != nil {
		return err
	}

	if options.quiet {
		return printImageIDs(dockerCli, images)
	}
	return printImages(dockerCli, images, options)
}

func getImageDescriptors(bundleStore store.BundleStore) ([]imageDesc, error) {
	references, err := bundleStore.List()
	if err != nil {
		return nil, err
	}
	images := make([]imageDesc, len(references))
	for i, ref := range references {
		b, err := bundleStore.Read(ref)
		if err != nil {
			return nil, err
		}

		images[i] = getImageDesc(b, ref)
	}
	return images, nil
}

func printImages(dockerCli command.Cli, list []imageDesc, options imageListOption) error {
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

	w := tabwriter.NewWriter(dockerCli.Out(), 0, 0, 1, ' ', 0)
	printHeaders(w, options.digests)
	for _, desc := range list {
		desc.println(w, options.digests)
	}

	return w.Flush()
}

func printImageIDs(dockerCli command.Cli, refs []imageDesc) error {
	var buf bytes.Buffer
	for _, ref := range refs {
		fmt.Fprintln(&buf, ref.ID)
	}
	fmt.Fprint(dockerCli.Out(), buf.String())
	return nil
}

func getImageID(bundle *relocated.Bundle, ref reference.Reference) (string, error) {
	id, ok := ref.(store.ID)
	if !ok {
		var err error
		id, err = store.FromBundle(bundle)
		if err != nil {
			return "", err
		}
	}
	return stringid.TruncateID(id.String()), nil
}

func printHeaders(w io.Writer, digests bool) {
	headers := []string{"REPOSITORY", "TAG"}
	if digests {
		headers = append(headers, "DIGEST")
	}
	headers = append(headers, "APP IMAGE ID", "APP NAME", "CREATED")
	fmt.Fprintln(w, strings.Join(headers, "\t"))
}

type imageDesc struct {
	ID         string        `json:"id,omitempty"`
	Name       string        `json:"name,omitempty"`
	Repository string        `json:"repository,omitempty"`
	Tag        string        `json:"tag,omitempty"`
	Digest     string        `json:"digest,omitempty"`
	Created    time.Duration `json:"created,omitempty"`
}

func getImageDesc(bundle *relocated.Bundle, ref reference.Reference) imageDesc {
	var id string
	id, _ = getImageID(bundle, ref)
	var repository string
	if n, ok := ref.(reference.Named); ok {
		repository = reference.FamiliarName(n)
	}
	var tag string
	if t, ok := ref.(reference.Tagged); ok {
		tag = t.Tag()
	}
	var digest string
	if t, ok := ref.(reference.Digested); ok {
		digest = t.Digest().String()
	}
	var created time.Duration
	if payload, err := packager.CustomPayload(bundle.Bundle); err == nil {
		if createdPayload, ok := payload.(packager.CustomPayloadCreated); ok {
			created = time.Now().UTC().Sub(createdPayload.CreatedTime())
		}
	}
	return imageDesc{
		ID:         id,
		Name:       bundle.Name,
		Repository: repository,
		Tag:        tag,
		Digest:     digest,
		Created:    created,
	}
}

func (desc imageDesc) humanDuration() string {
	if desc.Created > 0 {
		return units.HumanDuration(desc.Created) + " ago"
	}
	return ""
}

func (desc imageDesc) println(w io.Writer, digests bool) {
	values := []string{}
	values = append(values, orNone(desc.Repository), orNone(desc.Tag))
	if digests {
		values = append(values, orNone(desc.Digest))
	}
	values = append(values, desc.ID, desc.Name, desc.humanDuration())
	fmt.Fprintln(w, strings.Join(values, "\t"))
}

func orNone(s string) string {
	if s != "" {
		return s
	}
	return "<none>"
}
