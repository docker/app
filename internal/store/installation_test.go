package store

import (
	"os"
	"testing"

	"github.com/docker/app/internal/relocated"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/cnab-to-oci/relocation"

	"github.com/deislabs/cnab-go/claim"
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

	expectedInstallation := &Installation{
		Claim: claim.Claim{
			Name: "installation-name",
		},
		Reference: "mybundle:mytag",
	}

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

func TestApplyingRelocationMap(t *testing.T) {
	installation, _ := NewInstallation("name", "reference", &relocated.Bundle{
		Bundle: &bundle.Bundle{
			InvocationImages: []bundle.InvocationImage{
				{
					BaseImage: bundle.BaseImage{
						Image: "localimage:1.0-invoc",
					},
				},
			},
			Images: map[string]bundle.Image{
				"svc1": {
					BaseImage: bundle.BaseImage{
						Image: "svc-1:local",
					},
				},
				"redis": {
					BaseImage: bundle.BaseImage{
						Image: "redis:latest",
					},
				},
				"hello": {
					BaseImage: bundle.BaseImage{
						Image: "http-echo",
					},
				},
			},
		},
		RelocationMap: relocation.ImageRelocationMap{
			"localimage:1.0-invoc": "docker.io/repo/app:tag@sha256:9f9426498125d4017bdbdc861451bd447b9cb6d0c7a790093a65f508f45a1dd4",
			"svc-1:local":          "docker.io/repo/app:tag@sha256:d14de6677360066fcb302892bf288b8a907fddb264f587189ea17e691284e58c",
			"redis:latest":         "docker.io/repo/app:tag@sha256:e6e61849494c55a096cb48f7efb262271a50e2452b6a9e3553cf26f519f01d23",
		},
	})

	expectedBundle := &bundle.Bundle{
		InvocationImages: []bundle.InvocationImage{
			{
				BaseImage: bundle.BaseImage{
					Image: "docker.io/repo/app:tag@sha256:9f9426498125d4017bdbdc861451bd447b9cb6d0c7a790093a65f508f45a1dd4",
				},
			},
		},
		Images: map[string]bundle.Image{
			"svc1": {
				BaseImage: bundle.BaseImage{
					Image: "docker.io/repo/app:tag@sha256:d14de6677360066fcb302892bf288b8a907fddb264f587189ea17e691284e58c",
				},
			},
			"redis": {
				BaseImage: bundle.BaseImage{
					Image: "docker.io/repo/app:tag@sha256:e6e61849494c55a096cb48f7efb262271a50e2452b6a9e3553cf26f519f01d23",
				},
			},
			"hello": {
				BaseImage: bundle.BaseImage{
					Image: "http-echo",
				},
			},
		},
	}

	assert.DeepEqual(t, expectedBundle, installation.Bundle)
}
