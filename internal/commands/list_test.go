package commands

import (
	"testing"
	"time"

	"github.com/deislabs/duffle/pkg/claim"
	"github.com/docker/app/internal/store"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestGetInstallationsSorted(t *testing.T) {
	tmpDir := fs.NewDir(t, "")
	defer tmpDir.Remove()
	appstore, err := store.NewApplicationStore(tmpDir.Path())
	assert.NilError(t, err)
	installationStore, err := appstore.InstallationStore("my-context")
	assert.NilError(t, err)
	now := time.Now()

	oldInstallation := &store.Installation{
		Claim: claim.Claim{
			Name:     "old-installation",
			Modified: now.Add(-1 * time.Hour),
		},
	}
	newInstallation := &store.Installation{
		Claim: claim.Claim{
			Name:     "new-installation",
			Modified: now,
		},
	}
	assert.NilError(t, installationStore.Store(newInstallation))
	assert.NilError(t, installationStore.Store(oldInstallation))

	installations, err := getInstallations("my-context", tmpDir.Path())
	assert.NilError(t, err)
	assert.Equal(t, len(installations), 2)
	// First installation is the last modified
	assert.Equal(t, installations[0].Name, "new-installation")
	assert.Equal(t, installations[1].Name, "old-installation")
}
