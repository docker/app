package store

import (
	"os"
	"testing"

	"github.com/deislabs/duffle/pkg/claim"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestStoreAndReadInstallation(t *testing.T) {
	// Initialize an installation store
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()
	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	installationStore, err := appstore.InstallationStore("my-context")
	assert.NilError(t, err)

	expectedInstallation := claim.Claim{Name: "installation-name"}

	// Store the installation
	err = installationStore.Store(expectedInstallation)
	assert.NilError(t, err)

	// Check the file exists (my-context is hashed)
	_, err = os.Stat(dockerConfigDir.Join("app", "installations", "60b9683c6c2b05b8adc06ff4d150b15a5c69d74c7a7ee35bd733df12861dd2b0", "installation-name.json"))
	assert.NilError(t, err)

	// Read the installation
	actualInstallation, err := installationStore.Read("installation-name")
	assert.NilError(t, err)
	assert.DeepEqual(t, expectedInstallation, actualInstallation)
}
