package store

import (
	"os"
	"testing"

	"github.com/deislabs/cnab-go/credentials"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestStoreAndReadCredentialSet(t *testing.T) {
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()
	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	credentialStore, err := appstore.CredentialStore("my-context")
	assert.NilError(t, err)

	expectedCreds := &credentials.CredentialSet{Name: "creds-name"}

	// Store the credentials
	err = credentialStore.Store(expectedCreds)
	assert.NilError(t, err)

	// Check the file exists (my-context is hashed)
	_, err = os.Stat(dockerConfigDir.Join("app", "credentials", "60b9683c6c2b05b8adc06ff4d150b15a5c69d74c7a7ee35bd733df12861dd2b0", "creds-name.yaml"))
	assert.NilError(t, err)

	// Load it
	actualCreds, err := credentialStore.Read("creds-name")
	assert.NilError(t, err)
	assert.DeepEqual(t, expectedCreds, actualCreds)
}
