package build

import (
	"testing"

	"github.com/docker/app/internal/packager"
	"github.com/docker/buildx/build"
	"gotest.tools/assert"
)

func Test_parseCompose(t *testing.T) {
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
				Tags: []string{"frontend"},
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := packager.Extract("testdata/" + tt.name)
			assert.NilError(t, err)
			got, _, err := parseCompose(app, "testdata", buildOptions{})
			assert.NilError(t, err)
			_, ok := got["dontwant"]
			assert.Assert(t, !ok, "parseCompose() should have excluded 'dontwant' service")
			opt, ok := got[tt.service]
			assert.Assert(t, ok, "parseCompose() error = %s not converted into a build.Options", tt.service)
			assert.DeepEqual(t, opt, tt.want)
		})
	}
}
