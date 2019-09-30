package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/types"
	"github.com/docker/app/types/parameters"
	yaml "gopkg.in/yaml.v2"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

const (
	validMeta = `version: "0.1"
name: my-app`
)

func TestSubstituteBracedParams(t *testing.T) {
	composeFile := `
version: "3.6"
services:
  front:
    ports:
     - "${front.port}:80"
`
	parameters := map[string]string{
		"front.port": "8080",
	}
	s, err := substituteParams(parameters, composeFile)
	assert.NilError(t, err)
	assert.Equal(t, s, `
version: "3.6"
services:
  front:
    ports:
     - "8080:80"
`)
}

func TestSubstituteNamedParams(t *testing.T) {
	composeFile := `
version: "3.6"
services:
  back:
    ports:
     - "$back.port:90"
`
	parameters := map[string]string{
		"back.port": "9000",
	}
	s, err := substituteParams(parameters, composeFile)
	assert.NilError(t, err)
	assert.Equal(t, s, `
version: "3.6"
services:
  back:
    ports:
     - "9000:90"
`)
}

func checkRenderError(t *testing.T, userParameters map[string]string, composeFile string, expectedError string) {
	metadata := strings.NewReader(validMeta)

	app := &types.App{Path: "my-app"}
	assert.NilError(t, types.Metadata(metadata)(app))
	assert.NilError(t, types.WithComposes(strings.NewReader(composeFile))(app))
	_, err := Render(app, userParameters, nil)
	assert.ErrorContains(t, err, expectedError)
}

func TestRenderFailOnDefaultParamValueInCompose(t *testing.T) {
	composeFile := `
version: "3.6"
services:
  front:
    ports:
     - "${front.port:-9090}:80"
`
	userParameters := map[string]string{
		"front.port": "4242",
	}
	checkRenderError(t, userParameters, composeFile, "The default value syntax of compose file is not supported in Docker App. "+
		"The characters ':' and '-' are not allowed in parameter names. Invalid parameter: ${front.port:-9090}.")

	composeFile = `
version: "3.6"
services:
	front:
	ports:
		- "${front.port-9090}:80"
	`
	checkRenderError(t, userParameters, composeFile, "The default value syntax of compose file is not supported in Docker App. "+
		"The characters ':' and '-' are not allowed in parameter names. Invalid parameter: ${front.port-9090}.")
	composeFile = `
version: "3.6"
services:
	front:
	ports:
		- "${front.port:?Error}:80"
	`
	checkRenderError(t, userParameters, composeFile, "The custom error message syntax of compose file is not supported in Docker App. "+
		"The characters ':' and '?' are not allowed in parameter names. Invalid parameter: ${front.port:?Error}.")

	composeFile = `
version: "3.6"
services:
	front:
	ports:
		- "${front.port?Error:unset variable}:80"
	`
	checkRenderError(t, userParameters, composeFile, "The custom error message syntax of compose file is not supported in Docker App. "+
		"The characters ':' and '?' are not allowed in parameter names. Invalid parameter: ${front.port?Error:unset variable}.")

}
func TestSubstituteMixedParams(t *testing.T) {
	composeFile := `
version: "3.6"
services:
  front:
    ports:
     - "${front.port}:80"
    deploy:
      replicas: ${front.deploy.replicas}
  back:
    ports:
     - "$back.port:90"
`
	parameters := map[string]string{
		"front.port":            "8080",
		"back.port":             "9000",
		"front.deploy.replicas": "3",
	}
	s, err := substituteParams(parameters, composeFile)
	assert.NilError(t, err)
	assert.Equal(t, s, `
version: "3.6"
services:
  front:
    ports:
     - "8080:80"
    deploy:
      replicas: 3
  back:
    ports:
     - "9000:90"
`)
}

func TestSkipDoubleDollarCase(t *testing.T) {
	composeFile := `
	version: "3.7"
	services:
	  front:
		command: $$dollar
`
	s, err := substituteParams(map[string]string{}, composeFile)
	assert.NilError(t, err)
	assert.Equal(t, s, `
	version: "3.7"
	services:
	  front:
		command: $$dollar
`)
}

func TestSubstituteMissingParameterValue(t *testing.T) {
	composeFile := `
	version: "3.7"
	services:
	  front:
		deploy:
		  replicas: ${myapp.nginx_replicas}
	  debug:
		ports:
		- $aport
`
	parameters := map[string]string{
		"aport": "10000",
	}
	_, err := substituteParams(parameters, composeFile)
	assert.ErrorContains(t, err, "Failed to set value for myapp.nginx_replicas. Value not found in parameters.")
}

func TestRenderEnabledFalse(t *testing.T) {
	for _, tc := range []interface{}{"false", "\"false\"", "\"! true\""} {
		composeFile := fmt.Sprintf(`
version: "3.7"
services:
  foo:
    image: busybox
    "x-enabled": %s
`, tc)
		c, err := render("foo.dockerapp", composeFile, nil)
		assert.NilError(t, err)
		assert.Check(t, is.Len(c.Services, 0), fmt.Sprintf("Failed for %s", tc))
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
	composeFile := `
version: "3.6"
services:
  foo:
    image: busybox,
`
	c, err := render("foo.dockerapp", composeFile, map[string]bundle.Image{
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
