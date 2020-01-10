package converter

import (
	"github.com/cnabio/cnab-go/bundle"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/go/canonical/json"
	digest "github.com/opencontainers/go-digest"
	ocischema "github.com/opencontainers/image-spec/specs-go"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	// CNABConfigMediaType is the config media type of the CNAB config image manifest
	CNABConfigMediaType = "application/vnd.cnab.config.v1+json"
)

// PreparedBundleConfig contains the config blob, image manifest (and fallback), and descriptors for a CNAB config
type PreparedBundleConfig struct {
	ConfigBlob           []byte
	ConfigBlobDescriptor ocischemav1.Descriptor
	Manifest             []byte
	ManifestDescriptor   ocischemav1.Descriptor
	Fallback             *PreparedBundleConfig
}

// PrepareForPush serializes a bundle config, generates its image manifest, and its manifest descriptor
func PrepareForPush(b *bundle.Bundle) (*PreparedBundleConfig, error) {
	blob, err := json.MarshalCanonical(b)
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

func nonOCIDescriptorOf(blob []byte) distribution.Descriptor {
	return distribution.Descriptor{
		MediaType: schema2.MediaTypeImageConfig,
		Size:      int64(len(blob)),
		Digest:    digest.FromBytes(blob),
	}
}

func prepareNonOCIBundleConfig(blob []byte) (*PreparedBundleConfig, error) {
	desc := nonOCIDescriptorOf(blob)
	man, err := schema2.FromStruct(schema2.Manifest{
		Versioned: schema2.SchemaVersion,
		// Add a descriptor for the configuration because some registries
		// require the layers property to be defined and non-empty
		Layers: []distribution.Descriptor{
			desc,
		},
		Config: desc,
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
