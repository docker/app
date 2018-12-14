package mock

import (
	"context"

	"github.com/deislabs/duffle/pkg/builder"
	"github.com/deislabs/duffle/pkg/duffle/manifest"
)

// Component represents a mock component
type Component struct {
}

// Name represents the name of a mock component
func (dc Component) Name() string {
	return "cnab"
}

// Type represents the type of a mock component
func (dc Component) Type() string {
	return "mock-type"
}

// URI represents the URI of the artefact of a mock component
func (dc Component) URI() string {
	return "mock-uri:1.0.0"
}

// Digest represents the digest of a mock component
func (dc Component) Digest() string {
	return "mock-digest"
}

// NewComponent returns a new mock component
func NewComponent(c *manifest.Component) *Component {
	return &Component{}
}

// PrepareBuild is no-op for a mock component
func (dc *Component) PrepareBuild(ctx *builder.Context) error {
	return nil
}

// Build is no-op for a mock component
func (dc Component) Build(ctx context.Context, app *builder.AppContext) error {
	return nil
}
