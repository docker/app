package kubernetes

import (
	"io/ioutil"

	"github.com/docker/cli/cli/context"
	"github.com/docker/cli/cli/context/store"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// ToStoreMeta converts the endpoint to the store format
func (e *EndpointMeta) ToStoreMeta() store.Metadata {
	meta := e.EndpointMetaBase.ToStoreMeta()
	if e.DefaultNamespace != "" {
		meta[defaultNamespaceKey] = e.DefaultNamespace
	}
	return meta
}

// Save saves the Kubernetes endpoint in a context store
func Save(s store.Store, endpoint Endpoint) error {
	ctxMeta, err := s.GetContextMetadata(endpoint.ContextName)
	switch {
	case store.IsErrContextDoesNotExist(err):
		ctxMeta = store.ContextMetadata{
			Endpoints: make(map[string]store.Metadata),
			Metadata:  make(store.Metadata),
		}
	case err != nil:
		return err
	}
	ctxMeta.Endpoints[KubernetesEndpointKey] = endpoint.ToStoreMeta()
	if err := s.CreateOrUpdateContext(endpoint.ContextName, ctxMeta); err != nil {
		return err
	}
	return s.ResetContextEndpointTLSMaterial(endpoint.ContextName, KubernetesEndpointKey, endpoint.TLSData.ToStoreTLSData())
}

// FromKubeConfig creates a Kubernetes endpoint from a Kubeconfig file
func FromKubeConfig(name, kubeconfig, kubeContext, namespaceOverride string) (Endpoint, error) {
	cfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{CurrentContext: kubeContext, Context: clientcmdapi.Context{Namespace: namespaceOverride}})
	ns, _, err := cfg.Namespace()
	if err != nil {
		return Endpoint{}, err
	}
	clientcfg, err := cfg.ClientConfig()
	if err != nil {
		return Endpoint{}, err
	}
	var ca, key, cert []byte
	if ca, err = readFileOrDefault(clientcfg.CAFile, clientcfg.CAData); err != nil {
		return Endpoint{}, err
	}
	if key, err = readFileOrDefault(clientcfg.KeyFile, clientcfg.KeyData); err != nil {
		return Endpoint{}, err
	}
	if cert, err = readFileOrDefault(clientcfg.CertFile, clientcfg.CertData); err != nil {
		return Endpoint{}, err
	}
	var tlsData *context.TLSData
	if ca != nil || cert != nil || key != nil {
		tlsData = &context.TLSData{
			CA:   ca,
			Cert: cert,
			Key:  key,
		}
	}
	return Endpoint{
		EndpointMeta: EndpointMeta{
			EndpointMetaBase: context.EndpointMetaBase{
				ContextName:   name,
				Host:          clientcfg.Host,
				SkipTLSVerify: clientcfg.Insecure,
			},
			DefaultNamespace: ns,
		},
		TLSData: tlsData,
	}, nil
}

func readFileOrDefault(path string, defaultValue []byte) ([]byte, error) {
	if path != "" {
		return ioutil.ReadFile(path)
	}
	return defaultValue, nil
}
