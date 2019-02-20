package converter

import (
	_ "crypto/sha256" // this ensures we can parse sha256 digests
	"encoding/json"
	"errors"
	"fmt"

	"github.com/containerd/containerd/images"
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/distribution/reference"
	ocischema "github.com/opencontainers/image-spec/specs-go"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const ( // General values
	// CNABVersion is the currently supported CNAB runtime version
	CNABVersion = "v1.0.0-WD"

	// OCIIndexSchemaVersion is the currently supported OCI index schema's version
	OCIIndexSchemaVersion = 1
)

// Type aliases to clarify to which annotation the values belong
type dockerAppFormatValue = string

type dockerTypeValue = string

type cnabDescriptorTypeValue = string

const ( // Top Level annotations and values
	// DockerAppFormatAnnotation is the top level annotation specifying the kind of the App Bundle
	DockerAppFormatAnnotation = "io.docker.app.format"
	// DockerAppFormatCNAB is the DockerAppFormatAnnotation value for CNAB
	DockerAppFormatCNAB dockerAppFormatValue = "cnab"

	// DockerTypeAnnotation is the annotation that designates the type of the application
	DockerTypeAnnotation = "io.docker.type"
	// DockerTypeApp is the value used to fill DockerTypeAnnotation when targeting a docker-app
	DockerTypeApp dockerTypeValue = "app"

	// CNABRuntimeVersionAnnotation is the top level annotation specifying the CNAB runtime version
	CNABRuntimeVersionAnnotation = "io.cnab.runtime_version"
	// CNABKeywordsAnnotation is the top level annotation specifying a list of keywords
	CNABKeywordsAnnotation = "io.cnab.keywords"
)

const ( // Descriptor level annotations and values
	// CNABDescriptorTypeAnnotation is a descriptor-level annotation specifying the type of reference image (currently invocation or component)
	CNABDescriptorTypeAnnotation = "io.cnab.type"
	// CNABDescriptorTypeInvocation is the CNABDescriptorTypeAnnotation value for invocation images
	CNABDescriptorTypeInvocation cnabDescriptorTypeValue = "invocation"
	// CNABDescriptorTypeComponent is the CNABDescriptorTypeAnnotation value for component images
	CNABDescriptorTypeComponent cnabDescriptorTypeValue = "component"
	// CNABDescriptorTypeConfig is the CNABDescriptorTypeAnnotation value for bundle configuration
	CNABDescriptorTypeConfig cnabDescriptorTypeValue = "config"

	// CNABDescriptorComponentNameAnnotation is a decriptor-level annotation specifying the component name
	CNABDescriptorComponentNameAnnotation = "io.cnab.component_name"

	// CNABDescriptorOriginalNameAnnotation is a decriptor-level annotation specifying the original image name
	CNABDescriptorOriginalNameAnnotation = "io.cnab.original_name"
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
func ConvertBundleToOCIIndex(b *bundle.Bundle, targetRef reference.Named, bundleConfigManifestRef ocischemav1.Descriptor) (*ocischemav1.Index, error) {
	annotations, err := makeAnnotations(b)
	if err != nil {
		return nil, err
	}
	manifests, err := makeManifests(b, targetRef, bundleConfigManifestRef)
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

// ConvertOCIIndexToBundle converts an OCI index to a CNAB bundle representation
func ConvertOCIIndexToBundle(ix *ocischemav1.Index, config *BundleConfig, originRepo reference.Named) (*bundle.Bundle, error) {
	b := &bundle.Bundle{
		Actions:     config.Actions,
		Credentials: config.Credentials,
		Parameters:  config.Parameters,
	}
	if err := parseTopLevelAnnotations(ix.Annotations, b); err != nil {
		return nil, err
	}
	if err := parseManifests(ix.Manifests, b, originRepo); err != nil {
		return nil, err
	}
	return b, nil
}

func makeAnnotations(b *bundle.Bundle) (map[string]string, error) {
	result := map[string]string{
		DockerAppFormatAnnotation:         DockerAppFormatCNAB,
		CNABRuntimeVersionAnnotation:      CNABVersion,
		ocischemav1.AnnotationTitle:       b.Name,
		ocischemav1.AnnotationVersion:     b.Version,
		ocischemav1.AnnotationDescription: b.Description,
		DockerTypeAnnotation:              DockerTypeApp,
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

func parseTopLevelAnnotations(annotations map[string]string, into *bundle.Bundle) error {
	var ok bool
	if into.Name, ok = annotations[ocischemav1.AnnotationTitle]; !ok {
		return errors.New("manifest is missing title annotation " + ocischemav1.AnnotationTitle)
	}
	if into.Version, ok = annotations[ocischemav1.AnnotationVersion]; !ok {
		return errors.New("manifest is missing version annotation " + ocischemav1.AnnotationVersion)
	}
	into.Description = annotations[ocischemav1.AnnotationDescription]
	if maintainersJSON, ok := annotations[ocischemav1.AnnotationAuthors]; ok {
		if err := json.Unmarshal([]byte(maintainersJSON), &into.Maintainers); err != nil {
			return fmt.Errorf("unable to parse maintainers: %s", err)
		}
	}
	if keywordsJSON, ok := annotations[CNABKeywordsAnnotation]; ok {
		if err := json.Unmarshal([]byte(keywordsJSON), &into.Keywords); err != nil {
			return fmt.Errorf("unable to parse keywords: %s", err)
		}
	}
	return nil
}

func makeManifests(b *bundle.Bundle, targetReference reference.Named, bundleConfigManifestReference ocischemav1.Descriptor) ([]ocischemav1.Descriptor, error) {
	if len(b.InvocationImages) != 1 {
		return nil, errors.New("only one invocation image supported")
	}
	if bundleConfigManifestReference.Annotations == nil {
		bundleConfigManifestReference.Annotations = map[string]string{}
	}
	bundleConfigManifestReference.Annotations[CNABDescriptorTypeAnnotation] = CNABDescriptorTypeConfig
	manifests := []ocischemav1.Descriptor{bundleConfigManifestReference}
	invocationImage, err := makeDescriptor(b.InvocationImages[0].BaseImage, targetReference)
	if err != nil {
		return nil, fmt.Errorf("invalid invocation image: %s", err)
	}
	invocationImage.Annotations = map[string]string{
		CNABDescriptorTypeAnnotation: CNABDescriptorTypeInvocation,
	}
	manifests = append(manifests, invocationImage)
	for name, img := range b.Images {
		image, err := makeDescriptor(img.BaseImage, targetReference)
		if err != nil {
			return nil, fmt.Errorf("invalid image: %s", err)
		}
		image.Annotations = map[string]string{
			CNABDescriptorTypeAnnotation:          CNABDescriptorTypeComponent,
			CNABDescriptorComponentNameAnnotation: name,
			CNABDescriptorOriginalNameAnnotation:  img.Description,
		}
		manifests = append(manifests, image)
	}
	return manifests, nil
}

func parseManifests(descriptors []ocischemav1.Descriptor, into *bundle.Bundle, originRepo reference.Named) error {
	for _, d := range descriptors {
		var imageType string
		switch d.MediaType {
		case ocischemav1.MediaTypeImageManifest, ocischemav1.MediaTypeImageIndex:
			imageType = "oci"
		case images.MediaTypeDockerSchema2Manifest, images.MediaTypeDockerSchema2ManifestList:
			imageType = "docker"
		default:
			return fmt.Errorf("unsupported manifest descriptor %q with mediatype %q", d.Digest, d.MediaType)
		}
		descriptorType, ok := d.Annotations[CNABDescriptorTypeAnnotation]
		if !ok {
			return fmt.Errorf("manifest descriptor %q has no CNAB descriptor type annotation %q", d.Digest, CNABDescriptorTypeAnnotation)
		}
		if descriptorType == CNABDescriptorTypeConfig {
			continue
		}
		// strip tag/digest from originRepo
		originRepo, err := reference.ParseNormalizedNamed(originRepo.Name())
		if err != nil {
			return fmt.Errorf("failed to create a digested reference for manifest descriptor %q: %s", d.Digest, err)
		}
		ref, err := reference.WithDigest(originRepo, d.Digest)
		if err != nil {
			return fmt.Errorf("failed to create a digested reference for manifest descriptor %q: %s", d.Digest, err)
		}
		refFamiliar := reference.FamiliarString(ref)
		switch descriptorType {
		// The current descriptor is an invocation image
		case CNABDescriptorTypeInvocation:
			into.InvocationImages = append(into.InvocationImages, bundle.InvocationImage{
				BaseImage: bundle.BaseImage{
					Image:     refFamiliar,
					ImageType: imageType,
					MediaType: d.MediaType,
					Size:      uint64(d.Size),
				},
			})
		// The current descriptor is a component image
		case CNABDescriptorTypeComponent:
			componentName, ok := d.Annotations[CNABDescriptorComponentNameAnnotation]
			if !ok {
				return fmt.Errorf("component name missing in descriptor %q", d.Digest)
			}
			originalName := d.Annotations[CNABDescriptorOriginalNameAnnotation]
			if into.Images == nil {
				into.Images = make(map[string]bundle.Image)
			}
			into.Images[componentName] = bundle.Image{
				Description: originalName,
				BaseImage: bundle.BaseImage{
					Image:     refFamiliar,
					ImageType: imageType,
					MediaType: d.MediaType,
					Size:      uint64(d.Size),
				},
			}
		default:
			return fmt.Errorf("invalid CNAB descriptor type %q in descriptor %q", descriptorType, d.Digest)
		}
	}
	return nil
}

func makeDescriptor(baseImage bundle.BaseImage, targetReference reference.Named) (ocischemav1.Descriptor, error) {
	named, err := reference.ParseNormalizedNamed(baseImage.Image)
	if err != nil {
		return ocischemav1.Descriptor{}, fmt.Errorf("image %q is not a valid image reference: %s", baseImage.Image, err)
	}
	if named.Name() != targetReference.Name() {
		return ocischemav1.Descriptor{}, fmt.Errorf("image %q is not in the same repository as %q", baseImage.Image, targetReference.String())
	}
	digested, ok := named.(reference.Digested)
	if !ok {
		return ocischemav1.Descriptor{}, fmt.Errorf("image %q is not a digested reference", baseImage.Image)
	}
	switch baseImage.MediaType {
	case ocischemav1.MediaTypeImageManifest:
	case images.MediaTypeDockerSchema2Manifest:
	case ocischemav1.MediaTypeImageIndex:
	case images.MediaTypeDockerSchema2ManifestList:
	default:
		return ocischemav1.Descriptor{}, fmt.Errorf("unsupported media type %q for image %q", baseImage.MediaType, baseImage.Image)
	}
	return ocischemav1.Descriptor{
		Digest:    digested.Digest(),
		MediaType: baseImage.MediaType,
		Size:      int64(baseImage.Size),
	}, nil
}
