package store

import (
	"testing"

	"github.com/docker/cli/cli/command"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestNewApplicationStoreInitializesDirectories(t *testing.T) {
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()

	// create a new store inside the docker configuration directory
	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	assert.Equal(t, appstore.path, dockerConfigDir.Join("app"))

	// an installation store is created per context
	_, err = appstore.InstallationStore("my-context", command.OrchestratorSwarm)
	assert.NilError(t, err)

	// a credential store is created per context
	_, err = appstore.CredentialStore("my-context")
	assert.NilError(t, err)

	manifest := fs.Expected(
		t,
		fs.WithMode(0755),
		fs.WithDir("app",
			fs.WithDir("bundles"),
			fs.WithDir("credentials", fs.WithMode(0700),
				fs.WithDir("60b9683c6c2b05b8adc06ff4d150b15a5c69d74c7a7ee35bd733df12861dd2b0", fs.WithMode(0700))),
			fs.WithDir("installations",
				fs.WithDir("60b9683c6c2b05b8adc06ff4d150b15a5c69d74c7a7ee35bd733df12861dd2b0"))),
	)
	assert.Assert(t, fs.Equal(dockerConfigDir.Path(), manifest))
}
