package main

import (
	"fmt"
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
	"github.com/deislabs/duffle/pkg/utils/crud"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/types/parameters"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/store"
	cliopts "github.com/docker/cli/opts"
	"github.com/docker/docker/pkg/homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type installOptions struct {
	parametersOptions
	orchestrator     string
	namespace        string
	kubeNamespace    string
	stackName        string
	insecure         bool
	sendRegistryAuth bool
	targetContext    string
	credentialsets   []string
}

type parametersOptions struct {
	parametersFiles []string
	env             []string
}

func (o *parametersOptions) addFlags(flags *pflag.FlagSet) {
	flags.StringArrayVarP(&o.parametersFiles, "parameters-files", "f", []string{}, "Override parameters files")
	flags.StringArrayVarP(&o.env, "set", "s", []string{}, "Override parameters values")
}

type nameKind uint

const (
	_ nameKind = iota
	nameKindEmpty
	nameKindFile
	nameKindDir
	nameKindReference
)

const longDescription = `Install the application on either Swarm or Kubernetes.
Bundle name is optional, and can be
- empty: resolve to any *.dockerapp file or directory in working dir
- existing file path: work with any *.dockerapp file or dir, or any CNAB bundle file (signed or unsigned)
- match a bundle name in the local duffle bundle repository
- refers to a CNAB bundle in a container registry
`

// installCmd represents the install command
func installCmd(dockerCli command.Cli) *cobra.Command {
	var opts installOptions

	cmd := &cobra.Command{
		Use:     "install [<bundle name>] [options]",
		Aliases: []string{"deploy"},
		Short:   "Install an application",
		Long:    longDescription,
		Args:    cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstall(dockerCli, firstOrEmpty(args), opts)
		},
	}
	opts.parametersOptions.addFlags(cmd.Flags())
	cmd.Flags().StringVarP(&opts.orchestrator, "orchestrator", "o", "", "Orchestrator to install on (swarm, kubernetes)")
	cmd.Flags().StringVar(&opts.namespace, "namespace", "", "Namespace to use (default: namespace in metadata)")
	cmd.Flags().StringVar(&opts.kubeNamespace, "kubernetes-namespace", "default", "Kubernetes namespace to install into")
	cmd.Flags().StringVar(&opts.stackName, "name", "", "Installation name (defaults to application name)")
	cmd.Flags().StringVar(&opts.targetContext, "target-context", "", "Context on which to install the application")
	cmd.Flags().BoolVar(&opts.insecure, "insecure", false, "Use insecure registry, without SSL")
	cmd.Flags().BoolVar(&opts.sendRegistryAuth, "with-registry-auth", false, "Sends registry auth")
	cmd.Flags().StringArrayVarP(&opts.credentialsets, "credential-set", "c", []string{}, "Use a duffle credentialset (either a YAML file, or a credential set present in the duffle credential store)")

	return cmd
}

func runInstall(dockerCli command.Cli, appname string, opts installOptions) error {
	muteDockerCli(dockerCli)
	targetContext := getTargetContext(opts.targetContext)
	parameterValues, err := prepareParameters(opts.parametersOptions)
	if err != nil {
		return err
	}

	bndl, err := resolveBundle(dockerCli, opts.namespace, appname, opts.insecure)
	if err != nil {
		return err
	}
	if opts.sendRegistryAuth {
		return errors.New("with-registry-auth is not supported at the moment")
	}
	if err := bndl.Validate(); err != nil {
		return err
	}
	h := duffleHome()
	claimName := opts.stackName
	if claimName == "" {
		claimName = bndl.Name
	}
	claimStore := claim.NewClaimStore(crud.NewFileSystemStore(h.Claims(), "json"))
	if _, err = claimStore.Read(claimName); err == nil {
		return fmt.Errorf("installation %q already exists", claimName)
	}
	c, err := claim.New(claimName)
	if err != nil {
		return err
	}
	if _, ok := bndl.Parameters["docker.orchestrator"]; ok {
		parameterValues["docker.orchestrator"] = opts.orchestrator
	}
	if _, ok := bndl.Parameters["docker.kubernetes-namespace"]; ok {
		parameterValues["docker.kubernetes-namespace"] = opts.kubeNamespace
	}

	driverImpl, err := prepareDriver(dockerCli)
	if err != nil {
		return err
	}
	creds, err := prepareCredentialSet(targetContext, dockerCli.ContextStore(), bndl, opts.credentialsets)
	if err != nil {
		return err
	}
	if err := credentials.Validate(creds, bndl.Credentials); err != nil {
		return err
	}

	c.Bundle = bndl
	convertedParamValues := map[string]interface{}{}

	if err := applyParameterValues(parameterValues, bndl.Parameters, convertedParamValues); err != nil {
		return err
	}

	c.Parameters, err = bundle.ValuesOrDefaults(convertedParamValues, bndl)
	if err != nil {
		return err
	}
	inst := &action.Install{
		Driver: driverImpl,
	}
	err = inst.Run(c, creds, dockerCli.Out())
	err2 := claimStore.Store(*c)
	if err != nil {
		return fmt.Errorf("install failed: %v", err)
	}
	return err2
}

func applyParameterValues(parameterValues map[string]string, parameterDefinitions map[string]bundle.ParameterDefinition, finalValues map[string]interface{}) error {
	for k, v := range parameterValues {
		pd, ok := parameterDefinitions[k]
		if !ok {
			return fmt.Errorf("parameter %q is not defined in the bundle", k)
		}
		value, err := pd.ConvertValue(v)
		if err != nil {
			return errors.Wrapf(err, "invalid value for parameter %q", k)
		}
		if err := pd.ValidateParameterValue(value); err != nil {
			return errors.Wrapf(err, "invalid value for parameter %q", k)
		}
		finalValues[k] = value
	}
	return nil
}

func prepareParameters(opts parametersOptions) (map[string]string, error) {
	p, err := parameters.LoadFiles(opts.parametersFiles)
	if err != nil {
		return nil, err
	}
	d := cliopts.ConvertKVStringsToMap(opts.env)
	overrides, err := parameters.FromFlatten(d)
	if err != nil {
		return nil, err
	}
	if p, err = parameters.Merge(p, overrides); err != nil {
		return nil, err
	}
	return p.Flatten(), nil
}

func getAppNameKind(name string) (string, nameKind) {
	if name == "" {
		return name, nameKindEmpty
	}
	// name can be a bundle.json file, a single dockerapp file, or a dockerapp directory
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

func extractAndLoadAppBasedBundle(dockerCli command.Cli, namespace, name string) (*bundle.Bundle, error) {
	app, err := packager.Extract(name)
	if err != nil {
		return nil, err
	}
	defer app.Cleanup()
	return makeBundleFromApp(dockerCli, app, namespace, "")
}

func resolveBundle(dockerCli command.Cli, namespace, name string, insecure bool) (*bundle.Bundle, error) {
	// resolution logic:
	// - if there is a docker-app package in working directory, or an http:// / https:// prefix, use packager.Extract result
	// - the name has a .json or .cnab extension and refers to an existing file or web resource: load the bundle
	// - name matches a bundle name:version stored in duffle bundle store: use it
	// - pull the bundle from the registry and add it to the bundle store
	name, kind := getAppNameKind(name)
	switch kind {
	case nameKindFile:
		if strings.HasSuffix(name, internal.AppExtension) {
			return extractAndLoadAppBasedBundle(dockerCli, namespace, name)
		}
		return loader.NewDetectingLoader().Load(name)
	case nameKindDir, nameKindEmpty:
		return extractAndLoadAppBasedBundle(dockerCli, namespace, name)
	case nameKindReference:
		// TODO: pull the bundle
	}
	return nil, fmt.Errorf("could not resolve bundle %q", name)
}

func getTargetContext(optstargetContext string) string {
	var targetContext string
	switch {
	case optstargetContext != "":
		targetContext = optstargetContext
	case os.Getenv("DOCKER_TARGET_CONTEXT") != "":
		targetContext = os.Getenv("DOCKER_TARGET_CONTEXT")
	}
	if targetContext == "default" {
		targetContext = ""
	}
	return targetContext
}

func stringsKVToStringInterface(src map[string]string) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range src {
		result[k] = v
	}
	return result
}

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
		return nil, errors.New("no target context specified. use use --target-context= or DOCKER_TARGET_CONTEXT= to define it")
	}
	return creds, nil
}

func duffleHome() home.Home {
	if h := os.Getenv(home.HomeEnvVar); h != "" {
		return home.Home(h)
	}
	return home.Home(filepath.Join(homedir.Get(), ".duffle"))
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
