package image

import (
	"bufio"
	"bytes"
	"testing"
	"time"

	"github.com/docker/app/internal/store"
	"gotest.tools/fs"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/relocated"
	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution/reference"
	"gotest.tools/assert"
)

func TestListCmd(t *testing.T) {
	refs := []reference.Named{
		parseReference(t, "foo/bar@sha256:b59492bb814012ca3d2ce0b6728242d96b4af41687cc82166a4b5d7f2d9fb865"),
		parseReference(t, "foo/bar:1.0"),
		nil,
	}
	bundles := []relocated.Bundle{
		{
			Bundle: &bundle.Bundle{
				Name: "Digested App",
			},
		},
		{
			Bundle: &bundle.Bundle{
				Version:       "1.0.0",
				SchemaVersion: "1.0.0",
				Name:          "Foo App",
			},
		},
		{
			Bundle: &bundle.Bundle{
				Name: "Quiet App",
			},
		},
	}

	testCases := []struct {
		name           string
		expectedOutput string
		options        imageListOption
	}{
		{
			name: "TestList",
			expectedOutput: `REPOSITORY          TAG                 APP IMAGE ID        APP NAME            CREATED             
<none>              <none>              ad2828ea5653        Quiet App           N/A                 
foo/bar             1.0                 9aae408ee04f        Foo App             N/A                 
foo/bar             <none>              3f825b2d0657        Digested App        N/A                 
`,
			options: imageListOption{format: "table"},
		},
		{
			name: "TestTemplate",
			expectedOutput: `APP IMAGE ID        DIGEST
ad2828ea5653        <none>
9aae408ee04f        <none>
3f825b2d0657        sha256:b59492bb814012ca3d2ce0b6728242d96b4af41687cc82166a4b5d7f2d9fb865
`,
			options: imageListOption{format: "table {{.ID}}", digests: true},
		},
		{
			name: "TestListWithDigests",
			//nolint:lll
			expectedOutput: `REPOSITORY          TAG                 DIGEST                                                                    APP IMAGE ID        APP NAME                                CREATED             
<none>              <none>              <none>                                                                    ad2828ea5653        Quiet App                               N/A                 
foo/bar             1.0                 <none>                                                                    9aae408ee04f        Foo App                                 N/A                 
foo/bar             <none>              sha256:b59492bb814012ca3d2ce0b6728242d96b4af41687cc82166a4b5d7f2d9fb865   3f825b2d0657        Digested App                            N/A                 
`,
			options: imageListOption{format: "table", digests: true},
		},
		{
			name: "TestListWithQuiet",
			expectedOutput: `ad2828ea5653
9aae408ee04f
3f825b2d0657
`,
			options: imageListOption{format: "table", quiet: true},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testRunList(t, refs, bundles, tc.options, tc.expectedOutput)
		})
	}
}

func TestSortImages(t *testing.T) {
	images := []imageDesc{
		{ID: "1", Created: time.Date(2016, time.August, 15, 0, 0, 0, 0, time.UTC)},
		{ID: "2"},
		{ID: "3"},
		{ID: "4", Created: time.Date(2018, time.August, 15, 0, 0, 0, 0, time.UTC)},
		{ID: "5", Created: time.Date(2017, time.August, 15, 0, 0, 0, 0, time.UTC)},
	}
	sortImages(images)
	assert.Equal(t, "4", images[0].ID)
	assert.Equal(t, "5", images[1].ID)
	assert.Equal(t, "1", images[2].ID)
	assert.Equal(t, "2", images[3].ID)
	assert.Equal(t, "3", images[4].ID)
}

func parseReference(t *testing.T, s string) reference.Named {
	ref, err := reference.ParseNormalizedNamed(s)
	assert.NilError(t, err)
	return ref
}

func testRunList(t *testing.T, refs []reference.Named, bundles []relocated.Bundle, options imageListOption, expectedOutput string) {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	dockerCli, err := command.NewDockerCli(command.WithOutputStream(w))
	assert.NilError(t, err)
	bundleStore, err := store.NewBundleStore(fs.NewDir(t, "store").Path())
	assert.NilError(t, err)
	for i, ref := range refs {
		_, err = bundleStore.Store(&bundles[i], ref)
		assert.NilError(t, err)
	}
	err = runList(dockerCli, options, bundleStore)
	assert.NilError(t, err)
	w.Flush()
	actualOutput := buf.String()
	assert.Equal(t, actualOutput, expectedOutput)
}
