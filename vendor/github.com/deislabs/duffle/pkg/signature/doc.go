// Package signature provides signing tools for cryptographically signing bundles.
//
// These tools provide methods for marking authority, verifying authority, and extracting
// ciphertext.
//
// A Signer is used for signing things. A Verifier is used for taking a signed block and
// verifying it against a keyring. A Key represents a single key, while a KeyRing represents
// a collection of keys.
package signature
