package commands

import (
	"bytes"
	"testing"
	"time"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/claim"
	"github.com/docker/app/internal/store"
	appStore "github.com/docker/app/internal/store"
	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/golden"
)

func TestGetInstallationsSorted(t *testing.T) {
	tmpDir := fs.NewDir(t, "")
	defer tmpDir.Remove()
	appstore, err := appStore.NewApplicationStore(tmpDir.Path())
	assert.NilError(t, err)
	installationStore, err := appstore.InstallationStore("my-context")
	assert.NilError(t, err)
	now := time.Now()

	oldInstallation := &appStore.Installation{
		Claim: claim.Claim{
			Name:     "old-installation",
			Modified: now.Add(-1 * time.Hour),
		},
	}
	newInstallation := &appStore.Installation{
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

func TestGetInstallationsAllContexts(t *testing.T) {
	tmpDir := fs.NewDir(t, "")
	defer tmpDir.Remove()

	// Create one installation per context
	appstore, err := store.NewApplicationStore(tmpDir.Path())
	assert.NilError(t, err)

	installationStore1, err := appstore.InstallationStore("context1")
	assert.NilError(t, err)
	installation1 := &store.Installation{
		Claim: claim.Claim{
			Name:     "installation1",
			Created:  time.Now().Add(-24 * time.Hour),
			Modified: time.Now().Add(-24 * time.Hour),
			Result: claim.Result{
				Action: claim.ActionInstall,
				Status: claim.StatusSuccess,
			},
			Bundle: &bundle.Bundle{
				Name:    "Application",
				Version: "1.0.0",
			},
		},
		Reference: "user/application:1",
	}
	assert.NilError(t, installationStore1.Store(installation1))

	installationStore2, err := appstore.InstallationStore("context2")
	assert.NilError(t, err)
	installation2 := &store.Installation{
		Claim: claim.Claim{
			Name:     "installation2",
			Created:  time.Now().Add(-30 * 24 * time.Hour),
			Modified: time.Now().Add(-30 * 24 * time.Hour),
			Result: claim.Result{
				Action: claim.ActionUpgrade,
				Status: claim.StatusFailure,
			},
			Bundle: &bundle.Bundle{
				Name:    "OtherApplication",
				Version: "0.2.0",
			},
		},
		Reference: "myregistry.com/user/application:2",
	}
	assert.NilError(t, installationStore2.Store(installation2))

	out := bytes.NewBuffer(nil)
	err = printInstallations(out, tmpDir.Path(), []string{"context1", "context2"})
	assert.NilError(t, err)
	golden.Assert(t, out.String(), "list-all-contexts.golden")
}
