package build

import (
	"testing"

	"github.com/docker/distribution/reference"

	"github.com/docker/app/internal/packager"
	"github.com/docker/buildx/build"
	"gotest.tools/assert"
)

func Test_parseCompose(t *testing.T) {

	tag, err := reference.Parse("test:1.0")
	assert.NilError(t, err)

	tests := []struct {
		name    string
		service string
		want    build.Options
	}{
		{
			name:    "simple",
			service: "web",
			want: build.Options{
				Inputs: build.Inputs{
					ContextPath:    "testdata/web",
					DockerfilePath: "testdata/web/Dockerfile",
				},
				Tags: []string{"test:1.0-web"},
			},
		},
		{
			name:    "context",
			service: "web",
			want: build.Options{
				Inputs: build.Inputs{
					ContextPath:    "testdata/web",
					DockerfilePath: "testdata/web/Dockerfile.custom",
				},
				Tags: []string{"test:1.0-web"},
			},
		},
		{
			name:    "withargs",
			service: "web",
			want: build.Options{
				Inputs: build.Inputs{
					ContextPath:    "testdata/web",
					DockerfilePath: "testdata/web/Dockerfile",
				},
				BuildArgs: map[string]string{"foo": "bar"},
				Tags:      []string{"test:1.0-web"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := packager.Extract("testdata/" + tt.name)
			assert.NilError(t, err)
			got, err := parseCompose(app, "testdata", buildOptions{}, tag)
			assert.NilError(t, err)
			_, ok := got["dontwant"]
			assert.Assert(t, !ok, "parseCompose() should have excluded 'dontwant' service")
			opt, ok := got[tt.service]
			assert.Assert(t, ok, "parseCompose() error = %s not converted into a build.Options", tt.service)
			assert.DeepEqual(t, opt, tt.want)
		})
	}
}
