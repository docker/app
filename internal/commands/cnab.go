package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/deislabs/cnab-go/action"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/deislabs/cnab-go/driver"
	dockerDriver "github.com/deislabs/cnab-go/driver/docker"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	appstore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	contextstore "github.com/docker/cli/cli/context/store"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
)

type bindMount struct {
	required bool
	endpoint string
}

const defaultSocketPath string = "/var/run/docker.sock"

type credentialSetOpt func(b *bundle.Bundle, creds credentials.Set) error

func addNamedCredentialSets(credStore appstore.CredentialStore, namedCredentialsets []string) credentialSetOpt {
	return func(_ *bundle.Bundle, creds credentials.Set) error {
		for _, file := range namedCredentialsets {
			var (
				c   *credentials.CredentialSet
				err error
			)
			// Check the credentialset locally first, then try in the credential store
			if _, e := os.Stat(file); e == nil {
				c, err = credentials.Load(file)
			} else {
				c, err = credStore.Read(file)
				if os.IsNotExist(err) {
					err = e
				}
			}
			if err != nil {
				return err
			}
			values, err := c.Resolve()
			if err != nil {
				return err
			}
			if err := creds.Merge(values); err != nil {
				return err
			}
		}
		return nil
	}
}

func parseCommandlineCredential(c string) (string, string, error) {
	split := strings.SplitN(c, "=", 2)
	if len(split) != 2 || split[0] == "" {
		return "", "", errors.Errorf("failed to parse %q as a credential name=value", c)
	}
	name := split[0]
	value := split[1]
	return name, value, nil
}

func addCredentials(strcreds []string) credentialSetOpt {
	return func(_ *bundle.Bundle, creds credentials.Set) error {
		for _, c := range strcreds {
			name, value, err := parseCommandlineCredential(c)
			if err != nil {
				return err
			}
			if err := creds.Merge(credentials.Set{
				name: value,
			}); err != nil {
				return err
			}
		}
		return nil
	}
}

func addDockerCredentials(contextName string, store contextstore.Store) credentialSetOpt {
	// docker desktop contexts require some rewriting for being used within a container
	store = dockerDesktopAwareStore{Store: store}
	return func(_ *bundle.Bundle, creds credentials.Set) error {
		if contextName != "" {
			data, err := ioutil.ReadAll(contextstore.Export(contextName, store))
			if err != nil {
				return err
			}
			creds[internal.CredentialDockerContextName] = string(data)
		}
		return nil
	}
}

func addRegistryCredentials(shouldPopulate bool, dockerCli command.Cli) credentialSetOpt {
	return func(b *bundle.Bundle, creds credentials.Set) error {
		if _, ok := b.Credentials[internal.CredentialRegistryName]; !ok {
			return nil
		}

		registryCreds := map[string]types.AuthConfig{}
		if shouldPopulate {
			for _, img := range b.Images {
				named, err := reference.ParseNormalizedNamed(img.Image)
				if err != nil {
					return err
				}
				info, err := registry.ParseRepositoryInfo(named)
				if err != nil {
					return err
				}
				key := registry.GetAuthConfigKey(info.Index)
				if _, ok := registryCreds[key]; !ok {
					registryCreds[key] = command.ResolveAuthConfig(context.Background(), dockerCli, info.Index)
				}
			}
		}
		registryCredsJSON, err := json.Marshal(registryCreds)
		if err != nil {
			return err
		}
		creds[internal.CredentialRegistryName] = string(registryCredsJSON)
		return nil
	}
}

func prepareCredentialSet(b *bundle.Bundle, opts ...credentialSetOpt) (map[string]string, error) {
	creds := map[string]string{}
	for _, op := range opts {
		if err := op(b, creds); err != nil {
			return nil, err
		}
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

// prepareDriver prepares a driver per the user's request.
func prepareDriver(dockerCli command.Cli, bindMount bindMount, stdout io.Writer) (driver.Driver, *bytes.Buffer) {
	d := &dockerDriver.Driver{}
	errBuf := bytes.NewBuffer(nil)
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

	// Load any driver-specific config out of the environment.
	driverCfg := map[string]string{}
	for env := range d.Config() {
		driverCfg[env] = os.Getenv(env)
	}
	d.SetConfig(driverCfg)

	return d, errBuf
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

func extractAndLoadAppBasedBundle(dockerCli command.Cli, name string) (*bundle.Bundle, string, error) {
	app, err := packager.Extract(name)
	if err != nil {
		return nil, "", err
	}
	defer app.Cleanup()
	bndl, err := makeBundleFromApp(dockerCli, app, nil)
	return bndl, "", err
}

func loadBundleFromFile(filename string) (*bundle.Bundle, error) {
	b := &bundle.Bundle{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return b, err
	}
	return bundle.Unmarshal(data)
}

//resolveBundle looks for a CNAB bundle which can be in a Docker App Package format or
// a bundle stored locally or in the bundle store. It returns a built or found bundle,
// a reference to the bundle if it is found in the bundlestore, and an error.
func resolveBundle(dockerCli command.Cli, bundleStore appstore.BundleStore, name string, pullRef bool, insecureRegistries []string) (*bundle.Bundle, string, error) {
	// resolution logic:
	// - if there is a docker-app package in working directory, or an http:// / https:// prefix, use packager.Extract result
	// - the name has a .json or .cnab extension and refers to an existing file or web resource: load the bundle
	// - name matches a bundle name:version stored in the bundle store: use it
	// - pull the bundle from the registry and add it to the bundle store
	name, kind := getAppNameKind(name)
	switch kind {
	case nameKindFile:
		if pullRef {
			return nil, "", errors.Errorf("%s: cannot pull when referencing a file based app", name)
		}
		if strings.HasSuffix(name, internal.AppExtension) {
			return extractAndLoadAppBasedBundle(dockerCli, name)
		}
		bndl, err := loadBundleFromFile(name)
		return bndl, "", err
	case nameKindDir, nameKindEmpty:
		if pullRef {
			if kind == nameKindDir {
				return nil, "", errors.Errorf("%s: cannot pull when referencing a directory based app", name)
			}
			return nil, "", errors.Errorf("cannot pull when referencing a directory based app")
		}
		return extractAndLoadAppBasedBundle(dockerCli, name)
	case nameKindReference:
		ref, err := reference.ParseNormalizedNamed(name)
		if err != nil {
			return nil, "", errors.Wrap(err, name)
		}
		tagRef := reference.TagNameOnly(ref)
		bndl, err := bundleStore.LookupOrPullBundle(tagRef, pullRef, dockerCli.ConfigFile(), insecureRegistries)
		return bndl, tagRef.String(), err
	}
	return nil, "", fmt.Errorf("could not resolve bundle %q", name)
}

func requiredClaimBindMount(c claim.Claim, targetContextName string, dockerCli command.Cli) (bindMount, error) {
	var specifiedOrchestrator string
	if rawOrchestrator, ok := c.Parameters[internal.ParameterOrchestratorName]; ok {
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

	ctxMeta, err := s.GetMetadata(targetContextName)
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
	registryOpts registryOptions, pullOpts pullOptions, paramsOpts parametersOptions) (*action.RunCustom, *appstore.Installation, *bytes.Buffer, error) {
	s, err := appstore.NewApplicationStore(config.Dir())
	if err != nil {
		return nil, nil, nil, err
	}
	bundleStore, err := s.BundleStore()
	if err != nil {
		return nil, nil, nil, err
	}
	driverImpl, errBuf := prepareDriver(dockerCli, bindMount{}, stdout)
	bundle, ref, err := resolveBundle(dockerCli, bundleStore, appname, pullOpts.pull, registryOpts.insecureRegistries)
	if err != nil {
		return nil, nil, nil, err
	}
	installation, err := appstore.NewInstallation("custom-action", ref)
	if err != nil {
		return nil, nil, nil, err
	}
	installation.Bundle = bundle

	if err := mergeBundleParameters(installation,
		withFileParameters(paramsOpts.parametersFiles),
		withCommandLineParameters(paramsOpts.overrides),
		withStrictMode(paramsOpts.strictMode),
	); err != nil {
		return nil, nil, nil, err
	}

	a := &action.RunCustom{
		Action: actionName,
		Driver: driverImpl,
	}
	return a, installation, errBuf, nil
}

func isInstallationFailed(installation *appstore.Installation) bool {
	return installation.Result.Action == claim.ActionInstall &&
		installation.Result.Status == claim.StatusFailure
}
