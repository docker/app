package converter

import (
	"encoding/json"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/opencontainers/go-digest"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// BundleConfig describes a cnab bundle runtime config
type BundleConfig struct {
	SchemaVersion string                                `json:"schema_version" mapstructure:"schema_version"`
	Actions       map[string]bundle.Action              `json:"actions,omitempty" mapstructure:"actions,omitempty"`
	Parameters    map[string]bundle.ParameterDefinition `json:"parameters" mapstructure:"parameters"`
	Credentials   map[string]bundle.Location            `json:"credentials" mapstructure:"credentials"`
}

// CreateBundleConfig creates a bundle config from a CNAB
func CreateBundleConfig(b *bundle.Bundle) *BundleConfig {
	return &BundleConfig{
		SchemaVersion: CNABVersion,
		Actions:       b.Actions,
		Parameters:    b.Parameters,
		Credentials:   b.Credentials,
	}
}

// PrepareForPush serializes a bundle config, generates its image manifest, and its manifest descriptor
func (c *BundleConfig) PrepareForPush() (blob []byte, manifest []byte, blobDescriptor ocischemav1.Descriptor, manifestDescriptor ocischemav1.Descriptor, err error) {
	bytes, err := json.Marshal(c)
	if err != nil {
		return nil, nil, ocischemav1.Descriptor{}, ocischemav1.Descriptor{}, err
	}
	man, err := schema2.FromStruct(schema2.Manifest{
		Versioned: schema2.SchemaVersion,
		Config: distribution.Descriptor{
			MediaType: schema2.MediaTypeImageConfig,
			Size:      int64(len(bytes)),
			Digest:    digest.FromBytes(bytes),
		},
	})
	if err != nil {
		return nil, nil, ocischemav1.Descriptor{}, ocischemav1.Descriptor{}, err
	}
	manBytes, err := man.MarshalJSON()
	if err != nil {
		return nil, nil, ocischemav1.Descriptor{}, ocischemav1.Descriptor{}, err
	}
	return bytes,
		manBytes,
		ocischemav1.Descriptor{
			MediaType: schema2.MediaTypeImageConfig,
			Size:      int64(len(bytes)),
			Digest:    digest.FromBytes(bytes),
		},
		ocischemav1.Descriptor{
			MediaType: schema2.MediaTypeManifest,
			Digest:    digest.FromBytes(manBytes),
			Size:      int64(len(manBytes)),
		}, nil
}
