package render

import (
	"strings"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/types"
	"github.com/docker/app/types/parameters"
	composetypes "github.com/docker/cli/cli/compose/types"
	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

const (
	validMeta = `version: "0.1"
name: my-app`
)

func TestRenderMissingValue(t *testing.T) {
	configFiles := []composetypes.ConfigFile{
		{
			Config: map[string]interface{}{
				"version": "3",
				"services": map[string]interface{}{
					"foo": map[string]interface{}{
						"image": "${imageName}:${version}",
					},
				},
			},
		},
	}
	finalEnv := map[string]string{
		"imageName": "foo",
	}
	_, err := render("foo.dockerapp", configFiles, finalEnv, nil)
	assert.Check(t, err != nil)
	assert.Check(t, is.ErrorContains(err, "required variable"))
}

func TestRender(t *testing.T) {
	configFiles := []composetypes.ConfigFile{
		{
			Config: map[string]interface{}{
				"version": "3",
				"services": map[string]interface{}{
					"foo": map[string]interface{}{
						"image":   "busybox:${version}",
						"command": []interface{}{"-text", "${foo.bar}"},
					},
				},
			},
		},
	}
	finalEnv := map[string]string{
		"version": "latest",
		"foo.bar": "baz",
	}
	c, err := render("foo.dockerapp", configFiles, finalEnv, nil)
	assert.NilError(t, err)
	assert.Check(t, is.Len(c.Services, 1))
	assert.Check(t, is.Equal(c.Services[0].Image, "busybox:latest"))
	assert.Check(t, is.DeepEqual([]string(c.Services[0].Command), []string{"-text", "baz"}))
}

func TestRenderEnabledFalse(t *testing.T) {
	for _, tc := range []interface{}{false, "false", "! ${myapp.debug}"} {
		configs := []composetypes.ConfigFile{
			{
				Config: map[string]interface{}{
					"version": "3.7",
					"services": map[string]interface{}{
						"foo": map[string]interface{}{
							"image":     "busybox",
							"command":   []interface{}{"-text", "foo"},
							"x-enabled": tc,
						},
					},
				},
			},
		}
		c, err := render("foo.dockerapp", configs, map[string]string{
			"myapp.debug": "true",
		}, nil)
		assert.NilError(t, err)
		assert.Check(t, is.Len(c.Services, 0))
	}
}

func TestRenderUserParameters(t *testing.T) {
	metadata := strings.NewReader(validMeta)
	composeFile := strings.NewReader(`
version: "3.6"
services:
  front:
    image: wordpress
    ports:
     - "${front.port}:80"
    deploy:
      replicas: ${front.deploy.replicas}
  back:
    image: mysql
    ports:
     - "${back.port}:90"
`)
	parameters := strings.NewReader(`
front:
  deploy:
    replicas: 1
  port: 8484
back:
  port: 9090
`)
	app := &types.App{Path: "my-app"}
	assert.NilError(t, types.Metadata(metadata)(app))
	assert.NilError(t, types.WithComposes(composeFile)(app))
	assert.NilError(t, types.WithParameters(parameters)(app))
	userParameters := map[string]string{
		"front.deploy.replicas": "9",
		"front.port":            "4242",
		"back.port":             "6666",
	}
	c, err := Render(app, userParameters, nil)
	assert.NilError(t, err)
	s, err := yaml.Marshal(c)
	assert.NilError(t, err)
	assert.Equal(t, string(s), `version: "3.6"
services:
  back:
    image: mysql
    ports:
    - mode: ingress
      target: 90
      published: 6666
      protocol: tcp
  front:
    deploy:
      replicas: 9
    image: wordpress
    ports:
    - mode: ingress
      target: 80
      published: 4242
      protocol: tcp
`)
}

func TestRenderWithoutDefaultParameters(t *testing.T) {
	metadata := strings.NewReader(validMeta)
	composeFile := strings.NewReader(`
version: "3.6"
services:
  front:
    image: nginx
    deploy:
      replicas: ${nginx.replicas}
`)
	parameters := strings.NewReader("")
	app := &types.App{Path: "my-app"}
	assert.NilError(t, types.Metadata(metadata)(app))
	assert.NilError(t, types.WithComposes(composeFile)(app))
	assert.NilError(t, types.WithParameters(parameters)(app))
	userParameters := map[string]string{
		"nginx.replicas": "9",
	}
	c, err := Render(app, userParameters, nil)
	assert.NilError(t, err)
	s, err := yaml.Marshal(c)
	assert.NilError(t, err)
	assert.Equal(t, string(s), `version: "3.6"
services:
  front:
    deploy:
      replicas: 9
    image: nginx
`)
}

func TestValidateBrokenComposeFile(t *testing.T) {
	metadata := strings.NewReader(validMeta)
	brokenComposeFile := strings.NewReader(`version: "3.6"
unknown-property: value`)
	app := &types.App{Path: "my-app"}
	err := types.Metadata(metadata)(app)
	assert.NilError(t, err)
	err = types.WithComposes(brokenComposeFile)(app)
	assert.NilError(t, err)
	c, err := Render(app, nil, nil)
	assert.Assert(t, is.Nil(c))
	assert.Error(t, err, "failed to load Compose file: unknown-property Additional property unknown-property is not allowed")
}

func TestValidateRenderedApplication(t *testing.T) {
	metadata := strings.NewReader(validMeta)
	composeFile := strings.NewReader(`
version: "3.6"
services:
    hello:
        image: hashicorp/http-echo
        ports:
        - ${port}:${port}`)
	parameters := strings.NewReader(`port: 8080`)
	app := &types.App{Path: "my-app"}
	err := types.Metadata(metadata)(app)
	assert.NilError(t, err)
	err = types.WithComposes(composeFile)(app)
	assert.NilError(t, err)
	err = types.WithParameters(parameters)(app)
	assert.NilError(t, err)
	c, err := Render(app, nil, nil)
	assert.Assert(t, c != nil)
	assert.NilError(t, err)
}

func TestServiceImageOverride(t *testing.T) {
	configFiles := []composetypes.ConfigFile{
		{
			Config: map[string]interface{}{
				"version": "3",
				"services": map[string]interface{}{
					"foo": map[string]interface{}{
						"image": "busybox",
					},
				},
			},
		},
	}
	c, err := render("foo.dockerapp", configFiles, nil, map[string]bundle.Image{
		"foo": {BaseImage: bundle.BaseImage{Image: "test"}},
	})
	assert.NilError(t, err)
	assert.Check(t, is.Len(c.Services, 1))
	assert.Check(t, is.Equal(c.Services[0].Image, "test"))
}

func TestRenderShouldMergeNonUniformParameters(t *testing.T) {
	metadata := strings.NewReader(validMeta)
	composeFile := strings.NewReader(`
version: "3.6"
services:
  any:
    image: none/none
    environment:
      SSH_USER: ${ssh.user}
`)
	p := strings.NewReader(`
ssh.user: FILLME
`)
	parametersOverride := strings.NewReader(`
ssh:
  user: sirtea
`)
	app := &types.App{Path: "my-app"}
	assert.NilError(t, types.Metadata(metadata)(app))
	assert.NilError(t, types.WithComposes(composeFile)(app))
	assert.NilError(t, types.WithParameters(p, parametersOverride)(app))
	params := app.Parameters()
	assert.Equal(t, len(params), 1)
	assert.DeepEqual(t, params, parameters.Parameters{
		"ssh": map[string]interface{}{"user": "sirtea"},
	})
}
