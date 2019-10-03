package build

import (
	"bytes"
	"io"

	"github.com/docker/distribution/reference"
	"github.com/opencontainers/go-digest"
)

// computeDigest takes a bundle and produce a unigue reference.Digested
func computeDigest(bundle io.WriterTo) (reference.Named, error) {
	b := bytes.Buffer{}
	_, err := bundle.WriteTo(&b)
	if err != nil {
		return nil, err
	}
	digest := digest.SHA256.FromBytes(b.Bytes())
	ref := sha{digest}
	return ref, nil
}

type sha struct {
	d digest.Digest
}

var _ reference.Named = sha{""}
var _ reference.Digested = sha{""}

// Digest implement Digested.Digest()
func (s sha) Digest() digest.Digest {
	return s.d
}

// Digest implement Named.String()
func (s sha) String() string {
	return s.d.String()
}

// Digest implement Named.Name()
func (s sha) Name() string {
	return s.d.String()
}
