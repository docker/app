package image

import (
	"time"

	"github.com/docker/cli/cli/command/formatter"

	"github.com/docker/app/internal/packager"
	"github.com/docker/app/internal/relocated"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/stringid"
	"github.com/spf13/cobra"
)

type imageListOption struct {
	quiet   bool
	digests bool
	format  string
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
	cmd.Flags().StringVarP(&options.format, "format", "f", "table", "Format the output using the given syntax or Go template")
	cmd.Flags().SetAnnotation("format", "experimentalCLI", []string{"true"}) //nolint:errcheck

	return cmd
}

func runList(dockerCli command.Cli, options imageListOption, bundleStore store.BundleStore) error {
	images, err := getImageDescriptors(bundleStore)
	if err != nil {
		return err
	}

	ctx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewImageFormat(options.format, options.quiet, options.digests),
	}

	return ImageWrite(ctx, images)
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

type imageDesc struct {
	ID         string    `json:"id,omitempty"`
	Name       string    `json:"name,omitempty"`
	Repository string    `json:"repository,omitempty"`
	Tag        string    `json:"tag,omitempty"`
	Digest     string    `json:"digest,omitempty"`
	Created    time.Time `json:"created,omitempty"`
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
	var created time.Time
	if payload, err := packager.CustomPayload(bundle.Bundle); err == nil {
		if createdPayload, ok := payload.(packager.CustomPayloadCreated); ok {
			created = createdPayload.CreatedTime()
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
