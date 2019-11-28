package store

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/app/internal/image"

	"github.com/deislabs/cnab-go/bundle"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

func Test_storeByDigest(t *testing.T) {
	dockerConfigDir := fs.NewDir(t, t.Name(), fs.WithMode(0755))
	defer dockerConfigDir.Remove()
	appstore, err := NewApplicationStore(dockerConfigDir.Path())
	assert.NilError(t, err)
	imageStore, err := appstore.ImageStore()
	assert.NilError(t, err)

	bndl := image.FromBundle(&bundle.Bundle{Name: "bundle-name"})
	ref := parseRefOrDie(t, "test/simple:1.0")
	_, err = imageStore.Store(ref, bndl)
	assert.NilError(t, err)

	_, err = os.Stat(dockerConfigDir.Join("app", "bundles", "docker.io", "test", "simple", "_tags", "1.0", image.BundleFilename))
	assert.NilError(t, err)

	_, err = imageStore.Store(nil, bndl)
	assert.NilError(t, err)

	ids := dockerConfigDir.Join("app", "bundles", "_ids")
	infos, err := ioutil.ReadDir(ids)
	assert.NilError(t, err)
	assert.Equal(t, len(infos), 1)
	_, err = os.Stat(dockerConfigDir.Join("app", "bundles", "_ids", infos[0].Name(), image.BundleFilename))
	assert.NilError(t, err)
}

func TestFromString(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{
			name: "valid digest",
			args: "c661f4ad1e53d6825c65c01fc994793c3333542bc79c181f0acdc63aa908defc",
		},
		{
			name:    "invalid size",
			args:    "c661f4ad1e53d682",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			args:    "c661f4ad1e53d6825c65c01fc994793c3333542bc79c181f0acdc63a/foo/1.0",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromString(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil {
				assert.Equal(t, got.String(), tt.args)
			}
		})
	}
}
