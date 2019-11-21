package inspect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/claim"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/store"
	"github.com/docker/app/types"
	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/golden"
)

const (
	composeYAML = `version: "3.1"`
)

type inspectTestCase struct {
	name string
	args map[string]string
}

func TestImageInspect(t *testing.T) {
	dir := fs.NewDir(t, "inspect",
		fs.WithDir("no-maintainers",
			fs.WithFile(internal.ComposeFileName, composeYAML),
			fs.WithFile(internal.MetadataFileName, `
version: 0.1.0
name: myapp`),
			fs.WithFile(internal.ParametersFileName, ``),
		),
		fs.WithDir("no-description",
			fs.WithFile(internal.ComposeFileName, composeYAML),
			fs.WithFile(internal.MetadataFileName, `
version: 0.1.0
name: myapp
maintainers:
  - name: dev
    email: "dev@example.com"`),
			fs.WithFile(internal.ParametersFileName, ""),
		),
		fs.WithDir("no-parameters",
			fs.WithFile(internal.ComposeFileName, composeYAML),
			fs.WithFile(internal.MetadataFileName, `
version: 0.1.0
name: myapp
maintainers:
  - name: dev
    email: "dev@example.com"
description: "some description"`),
			fs.WithFile(internal.ParametersFileName, ""),
		),
		fs.WithDir("overridden",
			fs.WithFile(internal.ComposeFileName, `
version: "3.1"

services:
  web:
    image: nginx
    ports:
      - ${web.port}:80
`),
			fs.WithFile(internal.MetadataFileName, `
version: 0.1.0
name: myapp
`),
			fs.WithFile(internal.ParametersFileName, ""),
		),
		fs.WithDir("full",
			fs.WithFile(internal.ComposeFileName, `
version: "3.1"

services:
  web1:
    image: nginx:latest
    ports:
      - 8080-8100:12300-12320
    deploy:
      replicas: 2
  web2:
    image: nginx:latest
    ports:
      - 9080-9100:22300-22320
    deploy:
      replicas: 2
networks:
  my-network1:
  my-network2:
volumes:
  my-volume1:
  my-volume2:
secrets:
  my-secret1:
    file: ./my_secret1.txt
  my-secret2:
    file: ./my_secret2.txt
`),
			fs.WithFile(internal.MetadataFileName, `
version: 0.1.0
name: myapp
maintainers:
  - name: dev
    email: "dev@example.com"
description: "some description"`),
			fs.WithFile(internal.ParametersFileName, `
port: 8080
text: hello`),
			fs.WithFile("config.cfg", "something"),
		),
	)
	defer dir.Remove()

	t.Run("json", func(t *testing.T) {
		for _, testcase := range []inspectTestCase{
			{name: "no-maintainers"},
			{name: "no-description"},
			{name: "no-parameters"},
			{name: "overridden", args: map[string]string{"web.port": "80"}},
			{name: "full"},
		} {
			os.Setenv(internal.DockerInspectFormatEnvVar, "pretty")
			testImageInspect(t, dir, testcase, "")
			os.Setenv(internal.DockerInspectFormatEnvVar, "json")
			testImageInspect(t, dir, testcase, "-json")
		}
	})
}

func TestImageInspectCNABFormatJSON(t *testing.T) {
	testImageInspectCNAB(t, "json")
}

func TestImageInspectCNABFormatPretty(t *testing.T) {
	testImageInspectCNAB(t, "pretty")
}

func testImageInspectCNAB(t *testing.T, format string) {
	s := golden.Get(t, "bundle-json.golden")
	var bndl bundle.Bundle
	err := json.Unmarshal(s, &bndl)
	assert.NilError(t, err)

	expected := golden.Get(t, fmt.Sprintf("inspect-bundle-%s.golden", format))

	outBuffer := new(bytes.Buffer)
	err = ImageInspectCNAB(outBuffer, &bndl, format)
	assert.NilError(t, err)

	result := outBuffer.String()
	assert.Equal(t, string(expected), result)
}

func testImageInspect(t *testing.T, dir *fs.Dir, testcase inspectTestCase, suffix string) {
	app, err := types.NewAppFromDefaultFiles(dir.Join(testcase.name))
	assert.NilError(t, err)
	// Inspect twice to ensure output is stable (e.g. sorting of maps)
	for i := 0; i < 2; i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			outBuffer := new(bytes.Buffer)
			err = ImageInspect(outBuffer, app, testcase.args, nil)
			assert.NilError(t, err)
			golden.Assert(t, outBuffer.String(), fmt.Sprintf("inspect-%s%s.golden", testcase.name, suffix))
		})
	}
}

func getInstallation() store.Installation {
	created := time.Now().Add(time.Hour * -24)
	modified := time.Now().Add(time.Hour * -17)
	b := bundle.Bundle{
		SchemaVersion: "1.0.0",
		Name:          "hello-world",
		Version:       "0.1.0",
		Description:   "Hello, World!",
	}
	i := store.Installation{
		Claim: claim.Claim{
			Name:     "hello-world",
			Revision: "01DS2ZW4QKPXHTZXZ8YAP6S9W2",
			Created:  created,
			Modified: modified,
			Bundle:   &b,
			Result: claim.Result{
				Action: "upgrade",
				Status: "success",
			},
			Parameters: map[string]interface{}{
				"com.docker.app.args":                 "{}",
				"com.docker.app.inspect-format":       "json",
				"com.docker.app.kubernetes-namespace": "default",
				"com.docker.app.orchestrator":         "swarm",
				"com.docker.app.render-format":        "yaml",
				"com.docker.app.share-registry-creds": false,
				"port":                                "8080",
				"text":                                "Hello, World!",
			},
		},
		Reference: "docker.io/sirot/hello-world:0.1.0",
	}
	return i
}

func TestInspect(t *testing.T) {
	i := getInstallation()

	testCases := []struct {
		name   string
		format string
	}{
		{
			name:   "pretty",
			format: "pretty",
		},
		{
			name:   "json",
			format: "json",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var out bytes.Buffer
			err := Inspect(&out, &i, testCase.format)
			assert.NilError(t, err)
			golden.Assert(t, out.String(), fmt.Sprintf("inspect-app-%s.golden", testCase.name))
		})
	}
}
