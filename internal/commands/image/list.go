package image

import (
	"github.com/docker/app/internal/image"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/formatter"
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

			imageStore, err := appstore.ImageStore()
			if err != nil {
				return err
			}

			return runList(dockerCli, options, imageStore)
		},
	}
	flags := cmd.Flags()
	flags.BoolVarP(&options.quiet, "quiet", "q", false, "Only show numeric IDs")
	flags.BoolVarP(&options.digests, "digests", "", false, "Show image digests")
	cmd.Flags().StringVarP(&options.format, "format", "f", "table", "Format the output using the given syntax or Go template")
	cmd.Flags().SetAnnotation("format", "experimentalCLI", []string{"true"}) //nolint:errcheck

	return cmd
}

func runList(dockerCli command.Cli, options imageListOption, imageStore store.ImageStore) error {
	images, err := getImageDescriptors(imageStore)
	if err != nil {
		return err
	}

	ctx := formatter.Context{
		Output: dockerCli.Out(),
		Format: NewImageFormat(options.format, options.quiet, options.digests),
	}

	return Write(ctx, images)
}

func getImageDescriptors(imageStore store.ImageStore) ([]imageDesc, error) {
	references, err := imageStore.List()
	if err != nil {
		return nil, err
	}
	images := make([]imageDesc, len(references))
	for i, ref := range references {
		b, err := imageStore.Read(ref)
		if err != nil {
			return nil, err
		}

		images[i] = getImageDesc(b, ref)
	}
	return images, nil
}

func getImageID(bundle *image.AppImage, ref reference.Reference) (string, error) {
	id, ok := ref.(store.ID)
	if !ok {
		var err error
		id, err = store.FromAppImage(bundle)
		if err != nil {
			return "", err
		}
	}
	return stringid.TruncateID(id.String()), nil
}

type imageDesc struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	Digest     string `json:"digest,omitempty"`
}

func getImageDesc(bundle *image.AppImage, ref reference.Reference) imageDesc {
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
	return imageDesc{
		ID:         id,
		Name:       bundle.Name,
		Repository: repository,
		Tag:        tag,
		Digest:     digest,
	}
}
