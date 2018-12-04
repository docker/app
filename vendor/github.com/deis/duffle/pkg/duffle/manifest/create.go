package manifest

import (
	"io/ioutil"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

const runContent = `#!/bin/bash
action=$CNAB_ACTION

if [[ action == "install" ]]; then
echo "hey I am installing things over here"
elif [[ action == "uninstall" ]]; then
echo "hey I am uninstalling things now"
fi
`

const dockerfileContent = `FROM alpine:latest

COPY Dockerfile /cnab/Dockerfile
COPY app /cnab/app

CMD ["/cnab/app/run"]
`

// Scaffold takes a path and creates a minimal duffle manifest (duffle.yaml)
//  and scaffolds the components in that manifest
func Scaffold(path string) error {
	name := filepath.Base(path)
	m := &Manifest{
		Name:        name,
		Version:     "0.1.0",
		Description: "A short description of your bundle",
		Components: map[string]*Component{
			"cnab": {
				Name:    "cnab",
				Builder: "docker",
				Configuration: map[string]string{
					"registry": "microsoft",
				},
			},
		},
	}

	d, err := yaml.Marshal(m)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(path, "duffle.yaml"), d, 0644); err != nil {
		return err
	}
	cnabPath := filepath.Join(path, "cnab")
	if err := os.Mkdir(cnabPath, 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(cnabPath, "Dockerfile"), []byte(dockerfileContent), 0644); err != nil {
		return err
	}

	appPath := filepath.Join(cnabPath, "app")
	if err := os.Mkdir(appPath, 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(appPath, "run"), []byte(runContent), 0777); err != nil {
		return err
	}

	return nil
}
