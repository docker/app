package store

import (
	"bytes"
	"fmt"
	"io"
	"regexp"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
)

// ComputeDigest takes a bundle and produce a unigue reference.Digested
func ComputeDigest(bundle io.WriterTo) (digest.Digest, error) {
	b := bytes.Buffer{}
	_, err := bundle.WriteTo(&b)
	if err != nil {
		return "", err
	}
	return digest.SHA256.FromBytes(b.Bytes()), nil
}

func StringToRef(s string) (reference.Reference, error) {
	if named, err := reference.ParseNormalizedNamed(s); err == nil {
		return reference.TagNameOnly(named), nil
	}
	return FromString(s)
}

func FromString(s string) (ID, error) {
	if ok, _ := regexp.MatchString("[a-z0-9]{64}", s); !ok {
		return ID{}, fmt.Errorf("could not parse '%s' as a valid reference", s)
	}
	return ID{s}, nil
}

func FromBundle(bndle *bundle.Bundle) (ID, error) {
	digest, err := ComputeDigest(bndle)
	return ID{digest.Encoded()}, err
}

// ID is an unique identifier for docker app image bundle, implementing reference.Reference
type ID struct {
	digest string
}

var _ reference.Reference = ID{}

func (id ID) String() string {
	return id.digest
}
