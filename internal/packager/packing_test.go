package packager

import (
	"bytes"
	"os/user"
	"testing"

	"github.com/docker/docker/pkg/idtools"

	"github.com/docker/app/types"
	"github.com/docker/docker/pkg/archive"
	"gotest.tools/assert"
	"gotest.tools/fs"
)

func TestPackInvocationImageContext(t *testing.T) {
	app, err := types.NewAppFromDefaultFiles("testdata/packages/packing.dockerapp")
	assert.NilError(t, err)
	buf := bytes.NewBuffer(nil)
	assert.NilError(t, PackInvocationImageContext(app, buf))
	dir := fs.NewDir(t, t.Name())
	defer dir.Remove()
	usr, err := user.Current()
	assert.NilError(t, err)
	grp, err := user.LookupGroupId(usr.Gid)
	assert.NilError(t, err)
	// In an Unix-like OS, please make sure to have files /etc/sub{uid,gid} filled with:
	// <your_user_name>:<your_user_id>:1
	// and
	// <your_group>:<your_group_id>:1
	//
	// An example would be:
	// /etc/subuid -> myusr:1000:1
	// and
	// /etc/subgid -> mygrp:1000:1
	identityMapping, err := idtools.NewIdentityMapping(usr.Username, grp.Name)
	assert.NilError(t, err)
	assert.NilError(t, archive.Untar(buf, dir.Path(),
		&archive.TarOptions{
			NoLchown: true,
			UIDMaps:  identityMapping.UIDs(),
			GIDMaps:  identityMapping.GIDs(),
		}))
	expectedDir := fs.NewDir(t, t.Name(),
		fs.FromDir("testdata/packages"),
		fs.WithFile("Dockerfile", dockerFile),
		fs.WithFile(".dockerignore", dockerIgnore))
	defer expectedDir.Remove()
	assert.Assert(t, fs.Equal(dir.Path(), fs.ManifestFromDir(t, expectedDir.Path())))
}
