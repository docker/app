package commands

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/driver"
	"github.com/deislabs/duffle/pkg/duffle/home"
	"github.com/deislabs/duffle/pkg/loader"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	bundlestore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/pkg/errors"
)

type bindMount struct {
	required bool
	endpoint string
}

const defaultSocketPath string = "/var/run/docker.sock"

func prepareCredentialSet(contextName string, contextStore store.Store, b *bundle.Bundle, namedCredentialsets []string) (map[string]string, error) {
	// docker desktop contexts require some rewriting for being used within a container
	contextStore = dockerDesktopAwareStore{Store: contextStore}
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
		creds[internal.CredentialDockerContextName] = string(data)
	}
	_, requiresDockerContext := b.Credentials[internal.CredentialDockerContextName]
	_, hasDockerContext := creds[internal.CredentialDockerContextName]
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
func prepareDriver(dockerCli command.Cli, bindMount bindMount, stdout io.Writer) (driver.Driver, *bytes.Buffer, error) {
	driverImpl, err := driver.Lookup("docker")
	if err != nil {
		return driverImpl, nil, err
	}
	errBuf := bytes.NewBuffer(nil)
	if d, ok := driverImpl.(*driver.DockerDriver); ok {
		d.SetDockerCli(dockerCli)
		if stdout != nil {
			d.SetContainerOut(stdout)
		}
		d.SetContainerErr(errBuf)
		if bindMount.required {
			d.AddConfigurationOptions(func(config *container.Config, hostConfig *container.HostConfig) error {
				config.User = "0:0"
				mounts := []mount.Mount{
					{
						Type:   mount.TypeBind,
						Source: bindMount.endpoint,
						Target: bindMount.endpoint,
					},
				}
				hostConfig.Mounts = mounts
				return nil
			})
		}
	}

	// Load any driver-specific config out of the environment.
	if configurable, ok := driverImpl.(driver.Configurable); ok {
		driverCfg := map[string]string{}
		for env := range configurable.Config() {
			driverCfg[env] = os.Getenv(env)
		}
		configurable.SetConfig(driverCfg)
	}

	return driverImpl, errBuf, err
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

func resolveBundle(dockerCli command.Cli, name string, pullRef bool, insecureRegistries []string) (*bundle.Bundle, error) {
	// resolution logic:
	// - if there is a docker-app package in working directory, or an http:// / https:// prefix, use packager.Extract result
	// - the name has a .json or .cnab extension and refers to an existing file or web resource: load the bundle
	// - name matches a bundle name:version stored in duffle bundle store: use it
	// - pull the bundle from the registry and add it to the bundle store
	name, kind := getAppNameKind(name)
	switch kind {
	case nameKindFile:
		if pullRef {
			return nil, errors.Errorf("%s: cannot pull when referencing a file based app", name)
		}
		if strings.HasSuffix(name, internal.AppExtension) {
			return extractAndLoadAppBasedBundle(dockerCli, name)
		}
		return loader.NewDetectingLoader().Load(name)
	case nameKindDir, nameKindEmpty:
		if pullRef {
			if kind == nameKindDir {
				return nil, errors.Errorf("%s: cannot pull when referencing a directory based app", name)
			}
			return nil, errors.Errorf("cannot pull when referencing a directory based app")
		}
		return extractAndLoadAppBasedBundle(dockerCli, name)
	case nameKindReference:
		ref, err := reference.ParseNormalizedNamed(name)
		if err != nil {
			return nil, errors.Wrap(err, name)
		}
		return bundlestore.LookupOrPullBundle(dockerCli, reference.TagNameOnly(ref), pullRef, insecureRegistries)
	}
	return nil, fmt.Errorf("could not resolve bundle %q", name)
}

func requiredClaimBindMount(c claim.Claim, targetContextName string, dockerCli command.Cli) (bindMount, error) {
	var specifiedOrchestrator string
	if rawOrchestrator, ok := c.Parameters[internal.Namespace+"orchestrator"]; ok {
		specifiedOrchestrator = rawOrchestrator.(string)
	}

	return requiredBindMount(targetContextName, specifiedOrchestrator, dockerCli.ContextStore())
}

func requiredBindMount(targetContextName string, targetOrchestrator string, s store.Store) (bindMount, error) {
	if targetOrchestrator == "kubernetes" {
		return bindMount{}, nil
	}

	if targetContextName == "" {
		targetContextName = "default"
	}

	// in case of docker desktop, we want to rewrite the context in cases where it targets the local swarm or Kubernetes
	s = &dockerDesktopAwareStore{Store: s}

	ctxMeta, err := s.GetContextMetadata(targetContextName)
	if err != nil {
		return bindMount{}, err
	}
	dockerCtx, err := command.GetDockerContext(ctxMeta)
	if err != nil {
		return bindMount{}, err
	}
	if dockerCtx.StackOrchestrator == command.OrchestratorKubernetes {
		return bindMount{}, nil
	}
	dockerEndpoint, err := docker.EndpointFromContext(ctxMeta)
	if err != nil {
		return bindMount{}, err
	}

	host := dockerEndpoint.Host
	return bindMount{isDockerHostLocal(host), socketPath(host)}, nil
}

func socketPath(host string) string {
	if strings.HasPrefix(host, "unix://") {
		return strings.TrimPrefix(host, "unix://")
	}

	return defaultSocketPath
}

func isDockerHostLocal(host string) bool {
	return host == "" || strings.HasPrefix(host, "unix://") || strings.HasPrefix(host, "npipe://")
}

func prepareCustomAction(actionName string, dockerCli command.Cli, appname string, stdout io.Writer,
	registryOpts registryOptions, pullOpts pullOptions, paramsOpts parametersOptions) (*action.RunCustom, *claim.Claim, *bytes.Buffer, error) {

	c, err := claim.New("custom-action")
	if err != nil {
		return nil, nil, nil, err
	}
	driverImpl, errBuf, err := prepareDriver(dockerCli, bindMount{}, stdout)
	if err != nil {
		return nil, nil, nil, err
	}
	bundle, err := resolveBundle(dockerCli, appname, pullOpts.pull, registryOpts.insecureRegistries)
	if err != nil {
		return nil, nil, nil, err
	}
	c.Bundle = bundle

	parameters, err := mergeBundleParameters(c.Bundle,
		withFileParameters(paramsOpts.parametersFiles),
		withCommandLineParameters(paramsOpts.overrides),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	c.Parameters = parameters

	a := &action.RunCustom{
		Action: actionName,
		Driver: driverImpl,
	}
	return a, c, errBuf, nil
}
