package signature

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/openpgp/armor"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

// KeyRing represents a collection of keys as specified by OpenPGP
type KeyRing struct {
	entities          openpgp.EntityList
	PassphraseFetcher PassphraseFetcher
}

// Len returns the length of this keyring
//
// Length is the number of entitites stored in this ring.
func (r *KeyRing) Len() int {
	return len(r.entities)
}

// Add adds new keys to the keyring.
//
// Add is idempotent. If provided keys already exist, they will be
// silently ignored. This makes it easier to do bulk imports.
func (r *KeyRing) Add(keyReader io.Reader, armored bool) error {
	var entities openpgp.EntityList
	var err error
	if armored {
		entities, err = openpgp.ReadArmoredKeyRing(keyReader)
	} else {
		entities, err = openpgp.ReadKeyRing(keyReader)
	}
	if err != nil {
		return err
	}

	r.entities = append(r.entities, r.removeDuplicates(entities)...)
	return nil
}

// AddKey adds a *Key to the keyring.
//
// AddKey is idempotent. If a key exists already, it will be silently ignored.
func (r *KeyRing) AddKey(k *Key) {
	if r.isDuplicate(k.entity) {
		return
	}
	r.entities = append(r.entities, k.entity)
}

// removeDulicates filters out duplicate keys
func (r *KeyRing) removeDuplicates(entities []*openpgp.Entity) []*openpgp.Entity {
	remove := map[int]bool{}
	for i, e := range entities {
		if r.isDuplicate(e) {
			remove[i] = true
		}
	}

	if len(remove) == 0 {
		return entities
	}
	filtered := []*openpgp.Entity{}
	for i, e := range entities {
		if _, ok := remove[i]; !ok {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// isDuplicate compares the fingerprint of the given entity to all of the existing fingerprints
// If the fingerprint exists in the keyring, this returns true. Otherwise it returns false.
func (r *KeyRing) isDuplicate(e *openpgp.Entity) bool {
	for _, re := range r.entities {
		if re.PrimaryKey.Fingerprint == e.PrimaryKey.Fingerprint {
			return true
		}
	}
	return false
}

// Key returns the key with the given ID.
//
// ID is a hex ID or (conventionally) an email address.
//
// If no such key exists, this will return an error.
func (r *KeyRing) Key(id string) (*Key, error) {
	// NB: GnuPG allows any of the following to be used:
	// - Hex ID (we support)
	// - Email (we support)
	// - Substring match on OpenPGP User Name (we support if first two fail)
	// - Fingerprint
	// - OpenPGP User Name ("Name (Comment) <email>")
	// - Partial email
	// - Subject DN (x509)
	// - Issuer DN (x509)
	// - Keygrip (40 hex digits)

	hexID, err := strconv.ParseInt(id, 16, 64)
	if err == nil {
		k := r.entities.KeysById(uint64(hexID))
		l := len(k)
		if l > 1 {
			return nil, fmt.Errorf("required one key, got %d", l)
		}
		if l == 1 {
			return &Key{entity: k[0].Entity, PassphraseFetcher: r.PassphraseFetcher}, nil
		}
		// Else fallthrough and try a string-based lookup
	}

	// If we get here, there was no key found when looking by hex ID.
	// So we try again by string name in the email field. We also do weak matching
	// at the same time.
	weak := map[[20]byte]*openpgp.Entity{}
	for _, e := range r.entities {
		for _, ident := range e.Identities {
			// XXX Leave this commented section
			// It is not clear whether we should skip identities that were not self-signed
			// with the Sign flag on. Since the entity is at a higher level than the identity,
			// it seems like we are more interested in the entity's capability than the
			// identity the user requested, and we can always walk the subkeys to see if
			// any of those are allowed to sign. So I am leaving this commented.
			//if !ident.SelfSignature.FlagSign {
			//	continue
			//}
			if ident.UserId.Email == id {
				return &Key{entity: e, PassphraseFetcher: r.PassphraseFetcher}, nil
			}
			if strings.Contains(ident.Name, id) {
				weak[e.PrimaryKey.Fingerprint] = e
			}
		}
	}

	switch len(weak) {
	case 0:
		return nil, errors.New("key not found")
	case 1:
		for _, first := range weak {
			return &Key{entity: first, PassphraseFetcher: r.PassphraseFetcher}, nil
		}
	}
	return nil, errors.New("multiple matching keys found")
}

// PrivateKeys returns all of the private keys on this keyring.
//
// A private key is any key that has material in the private key packet. Note that
// this neither tests that the key is valid, nor decrypts an encrypted key.
func (r *KeyRing) PrivateKeys() []*Key {
	pks := []*Key{}
	for _, e := range r.entities {
		// This is the best test for a private key that I have been able to figure out.
		// It tests _if_ there is a PrivateKey on the entity. But that alone is insufficient,
		// since public keys sometimes have the public key data tacked on here. So then
		// we test for whether the private key has private key material OR whether it is
		// an encrypted key (which means it is private, but the data is not in the material
		// section until it has been decrypted).
		if e.PrivateKey != nil && (e.PrivateKey.PrivateKey != nil || e.PrivateKey.Encrypted) {
			pks = append(pks, &Key{
				entity:            e,
				PassphraseFetcher: r.PassphraseFetcher,
			})
		}
	}
	return pks
}

// Keys returns all keys (public and private).
func (r *KeyRing) Keys() []*Key {
	pks := []*Key{}
	for _, e := range r.entities {
		pks = append(pks, &Key{
			entity:            e,
			PassphraseFetcher: r.PassphraseFetcher,
		})
	}
	return pks
}

// SavePrivate writes a keyring to disk as a binary entity list.
//
// This is the standard format described by the OpenPGP specification. The file will thus be
// importable to any OpenPGP compliant app that can read entity lists (that is, a list of
// OpenPGP packets).
//
// Note that if the keyring contains encrypted keys, the saving process will need to
// decrypt every single key. Make sure the *KeyRing has a PassphraseFetcher before calling
// Save.
func (r *KeyRing) SavePrivate /*Ryan*/ (filepath string, clobber bool) error {
	if !clobber {
		if _, err := os.Stat(filepath); err == nil {
			return errors.New("keyring file exists")
		}
	}

	// Write to a buffer so we don't nuke a keychain.
	temp := bytes.NewBuffer(nil)
	for _, e := range r.entities {

		// The serializer has no decryption, so we have to do this manually before saving.
		// Yes, this is a major pain. But apparently encrypted keys cannot be serialized.
		if e.PrivateKey.Encrypted {
			if err := decryptPassphrase(e.PrimaryKey.KeyIdShortString(), e.PrivateKey, r.PassphraseFetcher); err != nil {
				return err
			}
		}

		for _, sk := range e.Subkeys {
			if sk.PrivateKey.Encrypted {
				if err := decryptPassphrase(e.PrimaryKey.KeyIdShortString()+" subkey", sk.PrivateKey, r.PassphraseFetcher); err != nil {
					return err
				}
			}
		}

		// According to the godocs, when we call this, we lose "signatures from other entities", but preserve public and private keys.
		if err := e.SerializePrivate(temp, nil); err != nil {
			return err
		}
	}

	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, temp)
	return err
}

// SavePublic saves the public keys into a file.
//
// Private key material is ignored.
func (r *KeyRing) SavePublic(filepath string, clobber, asciiArmor bool) error {
	if !clobber {
		if _, err := os.Stat(filepath); err == nil {
			return errors.New("keyring file exists")
		}
	}

	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write to a buffer so we don't nuke a keychain.
	temp := bytes.NewBuffer(nil)
	if err := r.SavePublicTo(temp, asciiArmor); err != nil {
		return err
	}
	_, err = io.Copy(f, temp)
	return err
}

// SavePublicTo saves the keyring to the given writer
//
// It removes private key material as it goes.
func (r *KeyRing) SavePublicTo(writer io.Writer, useArmor bool) error {
	w := writer
	if useArmor {
		headers := map[string]string{"Comment": "Duffle - https://cnab.io"}
		var err error
		w, err = armor.Encode(writer, openpgp.PublicKeyType, headers)
		if err != nil {
			return err
		}
		// Only close the writer if it is an encoder.
		defer w.(io.WriteCloser).Close()
	}
	for _, e := range r.entities {
		// According to the godocs, when we call this, we lose private key material, but keep public and signatures.
		// However, if I load a secret keyring (generated by GnuPG) and then serialize it, the private key
		// seems to still be there. Using `gpg --list-secret-keys`, I can recover secret keys after this method is run.
		if err := e.Serialize(w); err != nil {
			return err
		}
	}
	return nil
}

func decryptPassphrase(msg string, pk *packet.PrivateKey, fetcher PassphraseFetcher) error {
	if fetcher == nil {
		return errors.New("unable to decrypt key")
	}
	pass, err := fetcher(msg)
	if err != nil {
		return err
	}

	return pk.Decrypt(pass)
}

// LoadKeyRing loads a keyring from a path.
func LoadKeyRing(path string) (*KeyRing, error) {
	// TODO: Should we create a default passphrase fetcher?
	return LoadKeyRingFetcher(path, nil)
}

// LoadKeyRings loads multiple keyring files into one *KeyRing object
//
// This can be used to load both public and private keyrings for verification.
func LoadKeyRings(paths ...string) (*KeyRing, error) {
	if len(paths) == 0 {
		return &KeyRing{}, errors.New("no keyrings provided")
	}
	baseRing, err := LoadKeyRing(paths[0])
	if err != nil {
		return baseRing, err
	}
	for i := 1; i < len(paths); i++ {
		ring, err := LoadKeyRingFetcher(paths[i], baseRing.PassphraseFetcher)
		if err != nil {
			return baseRing, err
		}
		for _, k := range ring.Keys() {
			baseRing.AddKey(k)
		}
	}
	return baseRing, nil
}

// CreateKeyRing creates an empty in-memory keyring.
func CreateKeyRing(fetcher PassphraseFetcher) *KeyRing {
	return &KeyRing{
		entities:          openpgp.EntityList{},
		PassphraseFetcher: fetcher,
	}
}

// LoadKeyRingFetcher loads a keyring from a path.
//
// If PassphraseFetcher is non-nil, it will be called whenever an encrypted key needs to be decrypted.
// If left nil, this will cause the keyring to emit an error whenever an encrypted key needs to be decrypted.
func LoadKeyRingFetcher(path string, fetcher PassphraseFetcher) (*KeyRing, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entities, err := openpgp.ReadKeyRing(f)
	if err != nil {
		return nil, err
	}
	return &KeyRing{
		entities:          entities,
		PassphraseFetcher: fetcher,
	}, nil
}
