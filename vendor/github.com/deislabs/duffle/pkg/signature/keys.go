package signature

import (
	"bytes"
	"errors"
	"fmt"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

// PassphraseFetcher receives a keyname, and is responsible for returning the associated passphrase
type PassphraseFetcher func(name string) ([]byte, error)

// Key represents an individual signing key
//
// A key can be used to verify messages. If it also contains
// a private key, it can sign messages as well.
type Key struct {
	PassphraseFetcher PassphraseFetcher
	entity            *openpgp.Entity
	// selectedPrivateKey is reserved for use in cases where we want to
	// set a specific private key instead of looking through the entity
	// to load the key. This is necessary when choosing subkeys.
	selectedPrivateKey *packet.PrivateKey
}

var keyCreationConfig = packet.Config{
	RSABits: 3072, // Default keylength is only 2048. Following NIST recommendation for 3072.
}

// CreateKey creates a new key for the given user ID
//
// User ID should be in the form "NAME (COMMENT) <EMAIL>"
func CreateKey(user UserID) (*Key, error) {
	e, err := openpgp.NewEntity(user.Name, user.Comment, user.Email, &keyCreationConfig)
	if err != nil {
		return nil, err
	}

	// Okay, this is a little weird, but certain self-signing operations must be done before
	// a private key can be used. If we need to use this key before writing it to disk,
	// we'll need to do something like this:
	/*
		var buf bytes.Buffer
		if err := e.SerializePrivate(&buf, &keyCreationConfig); err != nil {
			return nil, err
		}
	*/

	return &Key{entity: e}, nil
}

// UserID returns the UserID for this key
//
// For OpenPGP insiders: This returns the FIRST identity that appears to have a valid name.
//
// An error is returned if no parseable user ID can be found.
func (k *Key) UserID() (UserID, error) {
	for i := range k.entity.Identities {
		id, err := ParseUserID(i)
		if err != nil {
			// Skip this one. No point in erroring out.
			continue
		}
		return id, nil
	}
	return UserID{}, errors.New("no parseable user identity attached to key")
}

// Fingerprint returns a string representation of the fingerprint.
func (k *Key) Fingerprint() string {
	fields := k.entity.PrimaryKey.Fingerprint
	var buf bytes.Buffer
	for i, b := range fields {
		if i > 0 && i%2 == 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(fmt.Sprintf("%0X", b))

	}
	return buf.String()
}

// CanSign indicates that a key is able to be used as a signer.
func (k *Key) CanSign() bool {
	_, err := k.findPrivateKey()
	return err == nil
}

// bestPrivateKey will find a private key and decrypt it if necessary.
//
// If a specific key is pinned on selectedPrivateKey, that key will be used.
// Otherwise, it will use the strategy in findPrivateKey.
func (k *Key) bestPrivateKey() (*packet.PrivateKey, error) {

	pk, err := k.findPrivateKey()
	if err != nil {
		return pk, err
	}

	// If key is not encrypted, return now.
	if !pk.Encrypted {
		return pk, nil
	}

	return pk, decryptPassphrase(k.entity.PrimaryKey.KeyIdShortString(), pk, k.PassphraseFetcher)
}

// findPrivateKey finds an acceptable private key for signing.
//
// If selectedPrivateKey is set this will use that key. Otherwise, it
// will start with the subkeys and seek for a signer, defaulting back to
// the top-level key.
//
// If no keys have the CanSign flag set, this will return an error.
//
// Finally, if no selectedPrivateKey is set, this will set the found
// key so that once it is unlocked we can avoid re-decrypting it.
func (k *Key) findPrivateKey() (*packet.PrivateKey, error) {
	// If a private key has already been set, use that.
	if k.selectedPrivateKey != nil {
		return k.selectedPrivateKey, nil
	}
	e := k.entity

	// It may be the case that a master key cannot be used for signing. It is not
	// clear how to test for that case. (in subkeys, you can do sk.Sig.FlagSign)
	if e.PrivateKey != nil && e.PrivateKey.CanSign() {
		k.selectedPrivateKey = e.PrivateKey
		return e.PrivateKey, nil
	}
	for _, sk := range e.Subkeys {
		// FlagSign checks if it is allowed to sign, while CanSign
		// verifies that the algorithm is capable of signing.
		if sk.Sig.FlagSign && sk.PrivateKey.CanSign() {
			k.selectedPrivateKey = sk.PrivateKey
			return sk.PrivateKey, nil
		}
	}

	return nil, errors.New("no signing key found")
}
