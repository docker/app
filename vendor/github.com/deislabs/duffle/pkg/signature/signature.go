package signature

import (
	"bytes"
	"crypto"
	"encoding/json"
	"errors"
	"io"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/clearsign"
	"golang.org/x/crypto/openpgp/packet"

	"github.com/deislabs/duffle/pkg/bundle"
)

// ErrNoSignature indicates that no signature was found in a block of text
var ErrNoSignature = errors.New("no signature block in data")

// Signer can sign bundles
//
// Signatures are OpenPGP Section 7 clearsigned blocks represented as ASCII-armored.
// To sign a bundle, the signer must be provided with the keys with which to sign.
//
// Signing is sensitive to whitespace. Thus, this package also takes on the responsibility
// of marshaling bundles into a canonical format before signing them.
//
// In addition to signing a bundle, the Signer can also attest an already-signed bundle.
// Attesting will calculate a detached signature on the same message body as the clearsigned
// representation.
type Signer struct {
	key    *Key
	Config *packet.Config
}

// NewSigner creates a new *Signer object.
// The key given here must be able to sign (create new signatures), which means it must point
// to a private key.
func NewSigner(key *Key) *Signer {
	return &Signer{
		key: key,
		Config: &packet.Config{
			DefaultHash: crypto.SHA256,
		},
	}
}

// Clearsign creates a new cleartext signature for the given bundle
//
// This creates a canonically generated bundle.json file, and then signs it. It is
// important that the format for the JSON file be identical each time.
func (s *Signer) Clearsign(b *bundle.Bundle) ([]byte, error) {
	sk, data, err := s.prepareSign(b)
	if err != nil {
		return data, err
	}

	// Clearsign the data
	return s.sign(data, sk)
}

// Attest generates an attestation (detached signature) for a signed bundle.
//
// This ONLY works on signed bundle files, and it requires the signed bundle
// as a []byte. It does not verify the signature on the signed block, nor does
// it parse the payload of the clearsigned block. Instead, it extracts the bytes
// from inside the block, and then re-signs that block, but with a detached
// signature.
//
// Where possible, the signature ought to be verified before it is attested.
// However, attestation does not require that the signature of the original
// block be validated.
func (s *Signer) Attest(signedBlock []byte) ([]byte, error) {
	empty := []byte{}
	block, _ := clearsign.Decode(signedBlock)
	if block == nil {
		return empty, ErrNoSignature
	}

	pk, err := s.key.bestPrivateKey()
	if err != nil {
		return empty, err
	}

	// We clearsign instead of using the openpgp.ArmoredDetachedSignText because the
	// later does not handle subkeys at all. It ONLY allows using the private key on
	// the main entity. Yet all the helper methods for that are unexported. Thus it
	// is more expedient to use the clearsign package and then extract the detached
	// signature from the block.
	signature, err := s.sign(block.Plaintext, pk)
	if err != nil {
		return empty, err
	}
	newblock, _ := clearsign.Decode(signature)
	if newblock == nil {
		return empty, errors.New("could not decode block just created")
	}

	body := bytes.NewBuffer(nil)
	if _, err := body.ReadFrom(newblock.ArmoredSignature.Body); err != nil {
		return empty, err
	}

	out := bytes.NewBuffer(nil)
	w, err := armor.Encode(out, openpgp.SignatureType, newblock.ArmoredSignature.Header)
	if err != nil {
		return empty, err
	}

	_, err = io.Copy(w, body)
	w.Close()
	if err != nil {
		return empty, err
	}

	return out.Bytes(), err
}

// prepareSign does work to prepare the bundle for signing.
func (s *Signer) prepareSign(b *bundle.Bundle) (*packet.PrivateKey, []byte, error) {
	res := []byte{}

	// We only proceed if we find at least one key that can be used to sign.
	pk, err := s.key.bestPrivateKey()
	if err != nil {
		return pk, res, err
	}

	// We want a canonical representation of a serialized bundle, which is why we
	// take the object.
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return pk, res, err
	}
	return pk, data, nil
}

// sign generates a clearsign of the text.
func (s *Signer) sign(data []byte, key *packet.PrivateKey) ([]byte, error) {
	res := []byte{}
	buf, dest := bytes.NewBuffer(data), bytes.NewBuffer(nil)
	out, err := clearsign.Encode(dest, key, s.Config)
	if err != nil {
		return res, err
	}
	if _, err := io.Copy(out, buf); err != nil {
		out.Close()
		return res, err
	}
	out.Close()
	return dest.Bytes(), nil
}
