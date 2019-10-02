package commands

import (
	"reflect"
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
		wantErr bool
	}{
		{
			name:    "simple",
			service: "web",
			want: build.Options{
				Inputs: build.Inputs{
					ContextPath:    "testdata/web",
					DockerfilePath: "testdata/web/Dockerfile",
				},
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

			got, err := parseCompose(app, buildOptions{})
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCompose() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			opt, ok := got[tt.service]
			if !ok {
				t.Errorf("parseCompose() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(opt, tt.want) {
				t.Errorf("parseCompose() got = %v, want = %v", opt, tt.want)
			}
		})
	}
}
