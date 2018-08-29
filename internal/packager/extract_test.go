package packager

import (
	"testing"

	"github.com/docker/distribution/reference"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func TestImageAppNameFromRef(t *testing.T) {
	refs := []struct {
		ref       string
		appName   string
		imageName string
	}{
		{ref: "foo", appName: "foo.dockerapp", imageName: "docker.io/library/foo.dockerapp"},
		{ref: "foo.dockerapp", appName: "foo.dockerapp", imageName: "docker.io/library/foo.dockerapp"},
		{ref: "foo:0.2.0", appName: "foo.dockerapp", imageName: "docker.io/library/foo.dockerapp:0.2.0"},
		{ref: "foo.dockerapp:0.2.0", appName: "foo.dockerapp", imageName: "docker.io/library/foo.dockerapp:0.2.0"},
		{ref: "namespace/foo", appName: "foo.dockerapp", imageName: "docker.io/namespace/foo.dockerapp"},
		{ref: "namespace/bar.dockerapp", appName: "bar.dockerapp", imageName: "docker.io/namespace/bar.dockerapp"},
		{ref: "namespace/bar:0.2.0", appName: "bar.dockerapp", imageName: "docker.io/namespace/bar.dockerapp:0.2.0"},
		{ref: "namespace/bar.dockerapp:0.2.0", appName: "bar.dockerapp", imageName: "docker.io/namespace/bar.dockerapp:0.2.0"},
		{ref: "gcr.io/namespace/baz", appName: "baz.dockerapp", imageName: "gcr.io/namespace/baz.dockerapp"},
		{ref: "gcr.io/namespace/baz.dockerapp", appName: "baz.dockerapp", imageName: "gcr.io/namespace/baz.dockerapp"},
		{ref: "gcr.io/namespace/baz:0.2.0", appName: "baz.dockerapp", imageName: "gcr.io/namespace/baz.dockerapp:0.2.0"},
		{ref: "gcr.io/namespace/baz.dockerapp:0.2.0", appName: "baz.dockerapp", imageName: "gcr.io/namespace/baz.dockerapp:0.2.0"},
	}
	for _, r := range refs {
		ref, err := reference.ParseNormalizedNamed(r.ref)
		assert.NilError(t, err)
		assert.Assert(t, is.Equal(imageNameFromRef(ref), r.imageName))
		assert.Assert(t, is.Equal(appNameFromRef(ref), r.appName))
	}
}
