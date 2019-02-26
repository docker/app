package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/driver"
	"github.com/deislabs/duffle/pkg/duffle/home"
	"github.com/deislabs/duffle/pkg/loader"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/store"
	"github.com/pkg/errors"
)

func prepareCredentialSet(contextName string, contextStore store.Store, b *bundle.Bundle, namedCredentialsets []string) (map[string]string, error) {
	creds := map[string]string{}
	for _, file := range namedCredentialsets {
		if _, err := os.Stat(file); err != nil {
			file = filepath.Join(duffleHome().Credentials(), file+".yaml")
		}
		c, err := credentials.Load(file)
		if err != nil {
			return nil, err
		}
		values, err := c.Resolve()
		if err != nil {
			return nil, err
		}
		for k, v := range values {
			if _, ok := creds[k]; ok {
				return nil, fmt.Errorf("ambiguous credential resolution: %q is present in multiple credential sets", k)
			}
			creds[k] = v
		}
	}
	if contextName != "" {
		data, err := ioutil.ReadAll(store.Export(contextName, contextStore))
		if err != nil {
			return nil, err
		}
		creds["docker.context"] = string(data)
	}
	_, requiresDockerContext := b.Credentials["docker.context"]
	_, hasDockerContext := creds["docker.context"]
	if requiresDockerContext && !hasDockerContext {
		return nil, errors.New("no target context specified. Use --target-context= or DOCKER_TARGET_CONTEXT= to define it")
	}
	return creds, nil
}

func getTargetContext(optstargetContext, currentContext string) string {
	var targetContext string
	switch {
	case optstargetContext != "":
		targetContext = optstargetContext
	case os.Getenv("DOCKER_TARGET_CONTEXT") != "":
		targetContext = os.Getenv("DOCKER_TARGET_CONTEXT")
	}
	if targetContext == "" {
		targetContext = currentContext
	}
	return targetContext
}

func duffleHome() home.Home {
	return home.Home(home.DefaultHome())
}

// prepareDriver prepares a driver per the user's request.
func prepareDriver(dockerCli command.Cli) (driver.Driver, error) {
	driverImpl, err := driver.Lookup("docker")
	if err != nil {
		return driverImpl, err
	}
	if d, ok := driverImpl.(*driver.DockerDriver); ok {
		d.SetDockerCli(dockerCli)
	}

	// Load any driver-specific config out of the environment.
	if configurable, ok := driverImpl.(driver.Configurable); ok {
		driverCfg := map[string]string{}
		for env := range configurable.Config() {
			driverCfg[env] = os.Getenv(env)
		}
		configurable.SetConfig(driverCfg)
	}

	return driverImpl, err
}

func getAppNameKind(name string) (string, nameKind) {
	if name == "" {
		return name, nameKindEmpty
	}
	// name can be a bundle.json or bundle.cnab file, a single dockerapp file, or a dockerapp directory
	st, err := os.Stat(name)
	if os.IsNotExist(err) {
		// try with .dockerapp extension
		st, err = os.Stat(name + internal.AppExtension)
		if err == nil {
			name += internal.AppExtension
		}
	}
	if err != nil {
		return name, nameKindReference
	}
	if st.IsDir() {
		return name, nameKindDir
	}
	return name, nameKindFile
}

func extractAndLoadAppBasedBundle(dockerCli command.Cli, name string) (*bundle.Bundle, error) {
	app, err := packager.Extract(name)
	if err != nil {
		return nil, err
	}
	defer app.Cleanup()
	return makeBundleFromApp(dockerCli, app)
}

func resolveBundle(dockerCli command.Cli, name string) (*bundle.Bundle, error) {
	// resolution logic:
	// - if there is a docker-app package in working directory, or an http:// / https:// prefix, use packager.Extract result
	// - the name has a .json or .cnab extension and refers to an existing file or web resource: load the bundle
	// - name matches a bundle name:version stored in duffle bundle store: use it
	// - pull the bundle from the registry and add it to the bundle store
	name, kind := getAppNameKind(name)
	switch kind {
	case nameKindFile:
		if strings.HasSuffix(name, internal.AppExtension) {
			return extractAndLoadAppBasedBundle(dockerCli, name)
		}
		return loader.NewDetectingLoader().Load(name)
	case nameKindDir, nameKindEmpty:
		return extractAndLoadAppBasedBundle(dockerCli, name)
	case nameKindReference:
		// TODO: pull the bundle
		fmt.Fprintln(dockerCli.Err(), "WARNING: pulling a CNAB is not yet supported")
	}
	return nil, fmt.Errorf("could not resolve bundle %q", name)
}
