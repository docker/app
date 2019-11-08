package converter

import (
	_ "crypto/sha256" // this ensures we can parse sha256 digests
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/containerd/containerd/images"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/cnab-to-oci/relocation"
	"github.com/docker/distribution/reference"
	ocischema "github.com/opencontainers/image-spec/specs-go"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const ( // General values
	// CNABVersion is the currently supported CNAB runtime version
	CNABVersion = "v1.0.0"

	// OCIIndexSchemaVersion is the currently supported OCI index schema's version
	OCIIndexSchemaVersion = 2
)

type cnabDescriptorTypeValue = string

const ( // Top Level annotations and values
	// CNABRuntimeVersionAnnotation is the top level annotation specifying the CNAB runtime version
	CNABRuntimeVersionAnnotation = "io.cnab.runtime_version"
	// CNABKeywordsAnnotation is the top level annotation specifying a list of keywords
	CNABKeywordsAnnotation = "io.cnab.keywords"
	// ArtifactTypeAnnotation is the top level annotation specifying the type of the artifact in the registry
	ArtifactTypeAnnotation = "org.opencontainers.artifactType"
	// ArtifactTypeValue is the value of ArtifactTypeAnnotion for CNAB bundles
	ArtifactTypeValue = "application/vnd.cnab.manifest.v1"
)

const ( // Descriptor level annotations and values
	// CNABDescriptorTypeAnnotation is a descriptor-level annotation specifying the type of reference image (currently invocation or component)
	CNABDescriptorTypeAnnotation = "io.cnab.manifest.type"
	// CNABDescriptorTypeInvocation is the CNABDescriptorTypeAnnotation value for invocation images
	CNABDescriptorTypeInvocation cnabDescriptorTypeValue = "invocation"
	// CNABDescriptorTypeComponent is the CNABDescriptorTypeAnnotation value for component images
	CNABDescriptorTypeComponent cnabDescriptorTypeValue = "component"
	// CNABDescriptorTypeConfig is the CNABDescriptorTypeAnnotation value for bundle configuration
	CNABDescriptorTypeConfig cnabDescriptorTypeValue = "config"

	// CNABDescriptorComponentNameAnnotation is a decriptor-level annotation specifying the component name
	CNABDescriptorComponentNameAnnotation = "io.cnab.component.name"
)

// GetBundleConfigManifestDescriptor returns the CNAB runtime config manifest descriptor from a OCI index
func GetBundleConfigManifestDescriptor(ix *ocischemav1.Index) (ocischemav1.Descriptor, error) {
	for _, d := range ix.Manifests {
		if d.Annotations[CNABDescriptorTypeAnnotation] == CNABDescriptorTypeConfig {
			return d, nil
		}
	}
	return ocischemav1.Descriptor{}, errors.New("bundle config not found")
}

// ConvertBundleToOCIIndex converts a CNAB bundle into an OCI Index representation
func ConvertBundleToOCIIndex(b *bundle.Bundle, targetRef reference.Named,
	bundleConfigManifestRef ocischemav1.Descriptor, relocationMap relocation.ImageRelocationMap) (*ocischemav1.Index, error) {
	annotations, err := makeAnnotations(b)
	if err != nil {
		return nil, err
	}
	manifests, err := makeManifests(b, targetRef, bundleConfigManifestRef, relocationMap)
	if err != nil {
		return nil, err
	}
	result := ocischemav1.Index{
		Versioned: ocischema.Versioned{
			SchemaVersion: OCIIndexSchemaVersion,
		},
		Annotations: annotations,
		Manifests:   manifests,
	}
	return &result, nil
}

// GenerateRelocationMap generates the bundle relocation map
func GenerateRelocationMap(ix *ocischemav1.Index, b *bundle.Bundle, originRepo reference.Named) (relocation.ImageRelocationMap, error) {
	relocationMap := relocation.ImageRelocationMap{}

	for _, d := range ix.Manifests {
		switch d.MediaType {
		case ocischemav1.MediaTypeImageManifest, ocischemav1.MediaTypeImageIndex:
		case images.MediaTypeDockerSchema2Manifest, images.MediaTypeDockerSchema2ManifestList:
		default:
			return nil, fmt.Errorf("unsupported manifest descriptor %q with mediatype %q", d.Digest, d.MediaType)
		}
		descriptorType, ok := d.Annotations[CNABDescriptorTypeAnnotation]
		if !ok {
			return nil, fmt.Errorf("manifest descriptor %q has no CNAB descriptor type annotation %q", d.Digest, CNABDescriptorTypeAnnotation)
		}
		if descriptorType == CNABDescriptorTypeConfig {
			continue
		}
		// strip tag/digest from originRepo
		originRepo, err := reference.ParseNormalizedNamed(originRepo.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to create a digested reference for manifest descriptor %q: %s", d.Digest, err)
		}
		ref, err := reference.WithDigest(originRepo, d.Digest)
		if err != nil {
			return nil, fmt.Errorf("failed to create a digested reference for manifest descriptor %q: %s", d.Digest, err)
		}
		refFamiliar := reference.FamiliarString(ref)
		switch descriptorType {
		// The current descriptor is an invocation image
		case CNABDescriptorTypeInvocation:
			if len(b.InvocationImages) == 0 {
				return nil, fmt.Errorf("unknown invocation image: %q", d.Digest)
			}
			relocationMap[b.InvocationImages[0].Image] = refFamiliar

		// The current descriptor is a component image
		case CNABDescriptorTypeComponent:
			componentName, ok := d.Annotations[CNABDescriptorComponentNameAnnotation]
			if !ok {
				return nil, fmt.Errorf("component name missing in descriptor %q", d.Digest)
			}
			c, ok := b.Images[componentName]
			if !ok {
				return nil, fmt.Errorf("component %q not found in bundle", componentName)
			}
			relocationMap[c.Image] = refFamiliar
		default:
			return nil, fmt.Errorf("invalid CNAB descriptor type %q in descriptor %q", descriptorType, d.Digest)
		}
	}

	return relocationMap, nil
}

func makeAnnotations(b *bundle.Bundle) (map[string]string, error) {
	result := map[string]string{
		CNABRuntimeVersionAnnotation:      b.SchemaVersion,
		ocischemav1.AnnotationTitle:       b.Name,
		ocischemav1.AnnotationVersion:     b.Version,
		ocischemav1.AnnotationDescription: b.Description,
		ArtifactTypeAnnotation:            ArtifactTypeValue,
	}
	if b.Maintainers != nil {
		maintainers, err := json.Marshal(b.Maintainers)
		if err != nil {
			return nil, err
		}
		result[ocischemav1.AnnotationAuthors] = string(maintainers)
	}
	if b.Keywords != nil {
		keywords, err := json.Marshal(b.Keywords)
		if err != nil {
			return nil, err
		}
		result[CNABKeywordsAnnotation] = string(keywords)
	}
	return result, nil
}

func makeManifests(b *bundle.Bundle, targetReference reference.Named,
	bundleConfigManifestReference ocischemav1.Descriptor, relocationMap relocation.ImageRelocationMap) ([]ocischemav1.Descriptor, error) {
	if len(b.InvocationImages) != 1 {
		return nil, errors.New("only one invocation image supported")
	}
	if bundleConfigManifestReference.Annotations == nil {
		bundleConfigManifestReference.Annotations = map[string]string{}
	}
	bundleConfigManifestReference.Annotations[CNABDescriptorTypeAnnotation] = CNABDescriptorTypeConfig
	manifests := []ocischemav1.Descriptor{bundleConfigManifestReference}
	invocationImage, err := makeDescriptor(b.InvocationImages[0].BaseImage, targetReference, relocationMap)
	if err != nil {
		return nil, fmt.Errorf("invalid invocation image: %s", err)
	}
	invocationImage.Annotations = map[string]string{
		CNABDescriptorTypeAnnotation: CNABDescriptorTypeInvocation,
	}
	manifests = append(manifests, invocationImage)
	images := makeSortedImages(b.Images)
	for _, name := range images {
		img := b.Images[name]
		image, err := makeDescriptor(img.BaseImage, targetReference, relocationMap)
		if err != nil {
			return nil, fmt.Errorf("invalid image: %s", err)
		}
		image.Annotations = map[string]string{
			CNABDescriptorTypeAnnotation:          CNABDescriptorTypeComponent,
			CNABDescriptorComponentNameAnnotation: name,
		}
		manifests = append(manifests, image)
	}
	return manifests, nil
}

func makeSortedImages(images map[string]bundle.Image) []string {
	var result []string
	for k := range images {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

func makeDescriptor(baseImage bundle.BaseImage, targetReference reference.Named, relocationMap relocation.ImageRelocationMap) (ocischemav1.Descriptor, error) {
	relocatedImage, ok := relocationMap[baseImage.Image]
	if !ok {
		return ocischemav1.Descriptor{}, fmt.Errorf("image %q not present in the relocation map", baseImage.Image)
	}

	named, err := reference.ParseNormalizedNamed(relocatedImage)
	if err != nil {
		return ocischemav1.Descriptor{}, fmt.Errorf("image %q is not a valid image reference: %s", relocatedImage, err)
	}
	if named.Name() != targetReference.Name() {
		return ocischemav1.Descriptor{}, fmt.Errorf("image %q is not in the same repository as %q", relocatedImage, targetReference.String())
	}
	digested, ok := named.(reference.Digested)
	if !ok {
		return ocischemav1.Descriptor{}, fmt.Errorf("image %q is not a digested reference", relocatedImage)
	}
	mediaType, err := getMediaType(baseImage, relocatedImage)
	if err != nil {
		return ocischemav1.Descriptor{}, err
	}
	if baseImage.Size == 0 {
		return ocischemav1.Descriptor{}, fmt.Errorf("image %q size is not set", relocatedImage)
	}

	return ocischemav1.Descriptor{
		Digest:    digested.Digest(),
		MediaType: mediaType,
		Size:      int64(baseImage.Size),
	}, nil
}

func getMediaType(baseImage bundle.BaseImage, relocatedImage string) (string, error) {
	mediaType := baseImage.MediaType
	if mediaType == "" {
		switch baseImage.ImageType {
		case "docker":
			mediaType = images.MediaTypeDockerSchema2Manifest
		case "oci":
			mediaType = ocischemav1.MediaTypeImageManifest
		default:
			return "", fmt.Errorf("unsupported image type %q for image %q", baseImage.ImageType, relocatedImage)
		}
	}
	switch mediaType {
	case ocischemav1.MediaTypeImageManifest:
	case images.MediaTypeDockerSchema2Manifest:
	case ocischemav1.MediaTypeImageIndex:
	case images.MediaTypeDockerSchema2ManifestList:
	default:
		return "", fmt.Errorf("unsupported media type %q for image %q", baseImage.MediaType, relocatedImage)
	}
	return mediaType, nil
}
