package converter

import (
	"encoding/json"

	"github.com/deislabs/cnab-go/bundle/definition"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	digest "github.com/opencontainers/go-digest"
	ocischema "github.com/opencontainers/image-spec/specs-go"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	// CNABConfigMediaType is the config media type of the CNAB config image manifest
	CNABConfigMediaType = "application/vnd.cnab.config.v1+json"
)

// BundleConfig describes a cnab bundle runtime config
type BundleConfig struct {
	SchemaVersion string                       `json:"schemaVersion" mapstructure:"schemaVersion"`
	Actions       map[string]bundle.Action     `json:"actions,omitempty" mapstructure:"actions,omitempty"`
	Definitions   definition.Definitions       `json:"definitions" mapstructure:"definitions"`
	Parameters    map[string]bundle.Parameter  `json:"parameters" mapstructure:"parameters"`
	Credentials   map[string]bundle.Credential `json:"credentials" mapstructure:"credentials"`
	Custom        map[string]interface{}       `json:"custom,omitempty" mapstructure:"custom"`
}

// PreparedBundleConfig contains the config blob, image manifest (and fallback), and descriptors for a CNAB config
type PreparedBundleConfig struct {
	ConfigBlob           []byte
	ConfigBlobDescriptor ocischemav1.Descriptor
	Manifest             []byte
	ManifestDescriptor   ocischemav1.Descriptor
	Fallback             *PreparedBundleConfig
}

// CreateBundleConfig creates a bundle config from a CNAB
func CreateBundleConfig(b *bundle.Bundle) *BundleConfig {
	return &BundleConfig{
		SchemaVersion: CNABVersion,
		Actions:       b.Actions,
		Definitions:   b.Definitions,
		Parameters:    b.Parameters,
		Credentials:   b.Credentials,
		Custom:        b.Custom,
	}
}

// PrepareForPush serializes a bundle config, generates its image manifest, and its manifest descriptor
func (c *BundleConfig) PrepareForPush() (*PreparedBundleConfig, error) {
	blob, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	fallbackChain := []bundleConfigPreparer{
		prepareOCIBundleConfig(CNABConfigMediaType),
		prepareOCIBundleConfig(ocischemav1.MediaTypeImageConfig),
		prepareNonOCIBundleConfig,
	}
	var first, current *PreparedBundleConfig
	for _, preparer := range fallbackChain {
		cfg, err := preparer(blob)
		if err != nil {
			return nil, err
		}
		if current == nil {
			first = cfg
		} else {
			current.Fallback = cfg
		}
		current = cfg
	}
	return first, nil
}

func descriptorOf(payload []byte, mediaType string) ocischemav1.Descriptor {
	return ocischemav1.Descriptor{
		MediaType: mediaType,
		Digest:    digest.FromBytes(payload),
		Size:      int64(len(payload)),
	}
}

type bundleConfigPreparer func(blob []byte) (*PreparedBundleConfig, error)

func prepareOCIBundleConfig(mediaType string) bundleConfigPreparer {
	return func(blob []byte) (*PreparedBundleConfig, error) {
		manifest := ocischemav1.Manifest{
			Versioned: ocischema.Versioned{
				SchemaVersion: OCIIndexSchemaVersion,
			},
			Config: descriptorOf(blob, mediaType),
		}
		manifestBytes, err := json.Marshal(&manifest)
		if err != nil {
			return nil, err
		}
		return &PreparedBundleConfig{
			ConfigBlob:           blob,
			ConfigBlobDescriptor: manifest.Config,
			Manifest:             manifestBytes,
			ManifestDescriptor:   descriptorOf(manifestBytes, ocischemav1.MediaTypeImageManifest),
		}, nil
	}
}

func prepareNonOCIBundleConfig(blob []byte) (*PreparedBundleConfig, error) {
	man, err := schema2.FromStruct(schema2.Manifest{
		Versioned: schema2.SchemaVersion,
		Config: distribution.Descriptor{
			MediaType: schema2.MediaTypeImageConfig,
			Size:      int64(len(blob)),
			Digest:    digest.FromBytes(blob),
		},
	})
	if err != nil {
		return nil, err
	}
	manBytes, err := man.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return &PreparedBundleConfig{
		ConfigBlob:           blob,
		ConfigBlobDescriptor: descriptorOf(blob, schema2.MediaTypeImageConfig),
		Manifest:             manBytes,
		ManifestDescriptor:   descriptorOf(manBytes, schema2.MediaTypeManifest),
	}, nil
}
