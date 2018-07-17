package packager

import (
    "testing"

    "gotest.tools/assert"
)

func TestSplitPackageName(t *testing.T) {
    ns, name := splitPackageName("foo/bar")
    assert.Equal(t, ns, "foo")
    assert.Equal(t, name, "bar")

    ns, name = splitPackageName("nonamespace")
    assert.Equal(t, ns, "")
    assert.Equal(t, name, "nonamespace")

    ns, name = splitPackageName("some.repo.tk/v3/foo/bar")
    assert.Equal(t, ns, "some.repo.tk/v3/foo")
    assert.Equal(t, name, "bar")
}
