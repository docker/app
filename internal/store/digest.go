package store

import (
	"bytes"
	"fmt"
	"io"
	"regexp"

	"github.com/docker/app/internal/image"

	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

var (
	identifierRegexp = regexp.MustCompile(`^([a-f0-9]{64})$`)

	shortIdentifierRegexp = regexp.MustCompile(`^([a-f0-9]{1,64})$`)
)

func isShortID(ref string) bool {
	return shortIdentifierRegexp.MatchString(ref)
}

// ComputeDigest takes a bundle and produce a unigue reference.Digested
func ComputeDigest(bundle io.WriterTo) (digest.Digest, error) {
	b := bytes.Buffer{}
	_, err := bundle.WriteTo(&b)
	if err != nil {
		return "", err
	}
	return digest.SHA256.FromBytes(b.Bytes()), nil
}

// StringToNamedRef converts a string to a named reference
func StringToNamedRef(s string) (reference.Named, error) {
	named, err := reference.ParseNormalizedNamed(s)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse %q as a valid reference", s)
	}
	return reference.TagNameOnly(named), nil
}

func FromString(s string) (ID, error) {
	if ok := identifierRegexp.MatchString(s); !ok {
		return "", fmt.Errorf("could not parse %q as a valid reference", s)
	}
	return fromID(s), nil
}

func fromID(s string) ID {
	digest := digest.NewDigestFromEncoded(digest.SHA256, s)
	return ID(digest)
}

func FromAppImage(img *image.AppImage) (ID, error) {
	digest, err := ComputeDigest(img)
	return ID(digest), err
}

// ID is an unique identifier for docker app image bundle, implementing reference.Reference
type ID digest.Digest

func (id ID) String() string {
	return id.Digest().Encoded()
}

func (id ID) Digest() digest.Digest {
	return digest.Digest(id)
}
