package signature

import (
	"bytes"
	"encoding/json"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"

	"github.com/deislabs/duffle/pkg/bundle"
)

// Verifier provides tools to verify a signed bundle.
// To verify a bundle, the signer may be given either a specific key, or a keyring.
//
// Because OpenPGP signing and verifying of cleartext is whitespace sensitive, this
// also provides tools to extract the exact text that was given.
type Verifier struct {
	keys *KeyRing
}

// NewVerifier creates a new *Verifier.
//
// They KeyRing is expected to contain all of the keys that are allowed to be used
// in verifying.
func NewVerifier(keyRing *KeyRing) *Verifier {
	return &Verifier{
		keys: keyRing,
	}
}

// Verify takes a signed bundle and verifies that the signature was signed by a known key
//
// This will return the key that the data was signed with.
func (v *Verifier) Verify(data []byte) (*Key, error) {
	// The second arg is the leftover text, which we don't care about.
	block, _ := clearsign.Decode(data)
	if block == nil {
		return nil, ErrNoSignature
	}
	ent, err := v.verifyBlock(block)
	return &Key{entity: ent}, err
}

// Extract verifies the signature against the keyring, and then returns the bundle
//
// This will return the bundle that was signed and the key it was signed with.
func (v *Verifier) Extract(data []byte) (*bundle.Bundle, *Key, error) {
	block, _ := clearsign.Decode(data)
	if block == nil {
		return nil, nil, ErrNoSignature
	}

	ent, err := v.verifyBlock(block)
	if err != nil {
		return nil, nil, err
	}
	res := &bundle.Bundle{}
	err = json.Unmarshal(block.Plaintext, res)
	return res, &Key{entity: ent}, err
}

// verifyBlock takes a block and verifies it as a detached signature.
func (v *Verifier) verifyBlock(block *clearsign.Block) (*openpgp.Entity, error) {
	buf := bytes.NewBuffer(block.Bytes)
	el := openpgp.KeyRing(v.keys.entities)
	return openpgp.CheckDetachedSignature(el, buf, block.ArmoredSignature.Body)
}
