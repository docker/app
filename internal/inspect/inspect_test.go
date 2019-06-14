package inspect

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal"
	"github.com/docker/app/types"
	composetypes "github.com/docker/cli/cli/compose/types"
	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/golden"
)

const (
	composeYAML = `version: "3.1"`
)

func TestInspect(t *testing.T) {
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

	for _, testcase := range []struct {
		name string
		args map[string]string
	}{
		{name: "no-maintainers"},
		{name: "no-description"},
		{name: "no-parameters"},
		{name: "overridden", args: map[string]string{"web.port": "80"}},
		{name: "full"},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			app, err := types.NewAppFromDefaultFiles(dir.Join(testcase.name))
			assert.NilError(t, err)
			// Inspect twice to ensure output is stable (e.g. sorting of maps)
			for i := 0; i < 2; i++ {
				t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
					outBuffer := new(bytes.Buffer)
					err = Inspect(outBuffer, app, testcase.args, nil)
					assert.NilError(t, err)
					golden.Assert(t, outBuffer.String(), fmt.Sprintf("inspect-%s.golden", testcase.name))
				})
			}

		})
	}
}

func TestInspectServices(t *testing.T) {
	for _, testcase := range []struct {
		name     string
		bundle   string
		expected string
	}{
		{
			name:     "pulled-app",
			bundle:   "testdata/sample-cnab/bundle-pulled.json",
			expected: "inspect-service-pulled.golden",
		},
		{
			name:     "new-app",
			bundle:   "testdata/sample-cnab/bundle-new.json",
			expected: "inspect-service-new.golden",
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			bundlefile, err := os.Open(testcase.bundle)
			assert.NilError(t, err)
			defer bundlefile.Close()
			b, err := bundle.ParseReader(bundlefile)
			assert.NilError(t, err)
			sc := composetypes.ServiceConfig{
				Name:  "hello",
				Image: b.Images["hello"].Image,
				Ports: []composetypes.ServicePortConfig{
					{
						Published: 8765,
					},
				},
			}
			services := []composetypes.ServiceConfig{sc}
			assert.NilError(t, err)
			outBuffer := new(bytes.Buffer)
			printServices(outBuffer, services, b.Images)
			assert.NilError(t, err)
			golden.Assert(t, outBuffer.String(), fmt.Sprintf(testcase.expected))
		})
	}
}
