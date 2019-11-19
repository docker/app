package image

import (
	"time"

	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/go-units"
)

const (
	defaultImageTableFormat           = "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Name}}\t{{if .CreatedSince }}{{.CreatedSince}}{{else}}N/A{{end}}\t"
	defaultImageTableFormatWithDigest = "table {{.Repository}}\t{{.Tag}}\t{{.Digest}}\t{{.ID}}{{.Name}}\t\t{{if .CreatedSince }}{{.CreatedSince}}{{else}}N/A{{end}}\t"

	imageIDHeader    = "APP ID"
	repositoryHeader = "REPOSITORY"
	tagHeader        = "TAG"
	digestHeader     = "DIGEST"
	imageNameHeader  = "APP NAME"
)

// NewImageFormat returns a format for rendering an ImageContext
func NewImageFormat(source string, quiet bool, digest bool) formatter.Format {
	switch source {
	case formatter.TableFormatKey:
		switch {
		case quiet:
			return formatter.DefaultQuietFormat
		case digest:
			return defaultImageTableFormatWithDigest
		default:
			return defaultImageTableFormat
		}
	}

	format := formatter.Format(source)
	if format.IsTable() && digest && !format.Contains("{{.Digest}}") {
		format += "\t{{.Digest}}"
	}
	return format
}

// Write writes the formatter images using the ImageContext
func Write(ctx formatter.Context, images []imageDesc) error {
	render := func(format func(subContext formatter.SubContext) error) error {
		return imageFormat(ctx, images, format)
	}
	return ctx.Write(newImageContext(), render)
}

func imageFormat(ctx formatter.Context, images []imageDesc, format func(subContext formatter.SubContext) error) error {
	for _, image := range images {
		img := &imageContext{
			trunc: ctx.Trunc,
			i:     image}
		if err := format(img); err != nil {
			return err
		}
	}
	return nil
}

type imageContext struct {
	formatter.HeaderContext
	trunc bool
	i     imageDesc
}

func newImageContext() *imageContext {
	imageCtx := imageContext{}
	imageCtx.Header = formatter.SubHeaderContext{
		"ID":           imageIDHeader,
		"Name":         imageNameHeader,
		"Repository":   repositoryHeader,
		"Tag":          tagHeader,
		"Digest":       digestHeader,
		"CreatedSince": formatter.CreatedSinceHeader,
	}
	return &imageCtx
}

func (c *imageContext) MarshalJSON() ([]byte, error) {
	return formatter.MarshalJSON(c)
}

func (c *imageContext) ID() string {
	if c.trunc {
		return stringid.TruncateID(c.i.ID)
	}
	return c.i.ID
}

func (c *imageContext) Name() string {
	if c.i.Name == "" {
		return "<none>"
	}
	return c.i.Name
}

func (c *imageContext) Repository() string {
	if c.i.Repository == "" {
		return "<none>"
	}
	return c.i.Repository
}

func (c *imageContext) Tag() string {
	if c.i.Tag == "" {
		return "<none>"
	}
	return c.i.Tag
}

func (c *imageContext) Digest() string {
	return c.i.Digest
}

func (c *imageContext) CreatedSince() string {
	if c.i.Created.IsZero() {
		return ""
	}
	return units.HumanDuration(time.Now().UTC().Sub(c.i.Created)) + " ago"
}
