package context

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/kubernetes"
	"github.com/docker/docker/pkg/homedir"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

type dockerEndpointOptions struct {
	host          string
	apiVersion    string
	ca            string
	cert          string
	key           string
	skipTLSVerify bool
	fromEnv       bool
}

func (o *dockerEndpointOptions) addFlags(flags *pflag.FlagSet, prefix string) {
	flags.StringVar(
		&o.host,
		prefix+"host",
		"",
		"required: specify the docker endpoint on witch to connect")
	flags.StringVar(
		&o.apiVersion,
		prefix+"api-version",
		"",
		"override negotiated api version")
	flags.StringVar(
		&o.ca,
		prefix+"tls-ca",
		"",
		"path to the ca file to validate docker endpoint")
	flags.StringVar(
		&o.cert,
		prefix+"tls-cert",
		"",
		"path to the cert file to authenticate to the docker endpoint")
	flags.StringVar(
		&o.key,
		prefix+"tls-key",
		"",
		"path to the key file to authenticate to the docker endpoint")
	flags.BoolVar(
		&o.skipTLSVerify,
		prefix+"tls-skip-verify",
		false,
		"skip tls verify when connecting to the docker endpoint")
	flags.BoolVar(
		&o.fromEnv,
		prefix+"from-env",
		false,
		"convert the current env-variable based configuration to a context")
}

func (o *dockerEndpointOptions) toEndpoint(cli command.Cli, contextName string) (docker.Endpoint, error) {
	if o.fromEnv {
		if cli.CurrentContext() != command.ContextDockerHost {
			return docker.Endpoint{},
				errors.New("cannot create a context from environment when a context is in use")
		}
		ep := cli.DockerEndpoint()
		ep.ContextName = contextName
		return ep, nil
	}
	tlsData, err := context.TLSDataFromFiles(o.ca, o.cert, o.key)
	if err != nil {
		return docker.Endpoint{}, err
	}
	return docker.Endpoint{
		EndpointMeta: docker.EndpointMeta{
			EndpointMetaBase: context.EndpointMetaBase{
				ContextName:   contextName,
				Host:          o.host,
				SkipTLSVerify: o.skipTLSVerify,
			},
			APIVersion: o.apiVersion,
		},
		TLSData: tlsData,
	}, nil
}

type kubernetesEndpointOptions struct {
	server            string
	ca                string
	cert              string
	key               string
	skipTLSVerify     bool
	defaultNamespace  string
	kubeconfigFile    string
	kubeconfigContext string
	fromEnv           bool
}

func (o *kubernetesEndpointOptions) addFlags(flags *pflag.FlagSet, prefix string) {
	flags.StringVar(
		&o.server,
		prefix+"host",
		"",
		"specify the kubernetes endpoint on witch to connect")
	flags.StringVar(
		&o.ca,
		prefix+"tls-ca",
		"",
		"path to the ca file to validate kubernetes endpoint")
	flags.StringVar(
		&o.cert,
		prefix+"tls-cert",
		"",
		"path to the cert file to authenticate to the kubernetes endpoint")
	flags.StringVar(
		&o.key,
		prefix+"tls-key",
		"",
		"path to the key file to authenticate to the kubernetes endpoint")
	flags.BoolVar(
		&o.skipTLSVerify,
		prefix+"tls-skip-verify",
		false,
		"skip tls verify when connecting to the kubernetes endpoint")
	flags.StringVar(
		&o.defaultNamespace,
		prefix+"default-namespace",
		"default",
		"override default namespace when connecting to kubernetes endpoint")
	flags.StringVar(
		&o.kubeconfigFile,
		prefix+"kubeconfig",
		"",
		"path to an existing kubeconfig file")
	flags.StringVar(
		&o.kubeconfigContext,
		prefix+"kubeconfig-context",
		"",
		fmt.Sprintf(`context to use in the kubeconfig file referenced in "%skubeconfig"`, prefix))
	flags.BoolVar(
		&o.fromEnv,
		prefix+"from-env",
		false,
		`use the default kubeconfig file or the value defined in KUBECONFIG environement variable`)
}

func (o *kubernetesEndpointOptions) toEndpoint(contextName string) (*kubernetes.Endpoint, error) {
	if o.kubeconfigFile == "" && o.fromEnv {
		if config := os.Getenv("KUBECONFIG"); config != "" {
			o.kubeconfigFile = config
		} else {
			o.kubeconfigFile = filepath.Join(homedir.Get(), ".kube/config")
		}
	}
	if o.kubeconfigFile != "" {
		ep, err := kubernetes.FromKubeConfig(
			contextName,
			o.kubeconfigFile,
			o.kubeconfigContext,
			o.defaultNamespace)
		if err != nil {
			return nil, err
		}
		return &ep, nil
	}
	if o.server != "" {
		tlsData, err := context.TLSDataFromFiles(o.ca, o.cert, o.key)
		if err != nil {
			return nil, err
		}
		return &kubernetes.Endpoint{
			EndpointMeta: kubernetes.EndpointMeta{
				EndpointMetaBase: context.EndpointMetaBase{
					ContextName:   contextName,
					Host:          o.server,
					SkipTLSVerify: o.skipTLSVerify,
				},
				DefaultNamespace: o.defaultNamespace,
			},
			TLSData: tlsData,
		}, nil
	}
	return nil, nil
}
