package crud

import (
	"os"
	"path/filepath"
	"strings"

	cnabCrud "github.com/deislabs/cnab-go/utils/crud"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	DefaultSecretLabelKey   = "installation_store"
	DefaultSecretLabelValue = "docker-app"

	// Kubernetes namespace where to store the secrets representing the installation claims
	DefaultKubernetesNamespace = "docker-app"
)

// NewKubernetesSecretsStore creates a Store backed by kubernetes secrets.
// Each key is represented by a secret in a kubernetes namespace.
func NewKubernetesSecretsStore(namespace string, label LabelKV) (cnabCrud.Store, error) {
	k8sClient, err := getClient()
	if err != nil {
		return nil, err
	}
	k8sStore := kubernetesSecretsStore{
		namespace: namespace,
		client:    k8sClient,
		labelKV:   label,
	}
	err = k8sStore.ensureNamespace()
	if err != nil {
		return nil, err
	}
	return k8sStore, nil
}

type LabelKV [2]string

func (l LabelKV) getKey() string {
	return l[0]
}

func (l LabelKV) getValue() string {
	return l[1]
}

func (l LabelKV) String() string {
	return strings.Join(l[:], "=")
}

type kubernetesSecretsStore struct {
	namespace string
	client    corev1.CoreV1Interface
	labelKV   LabelKV
}

func (s kubernetesSecretsStore) List() ([]string, error) {
	secrets, err := s.client.Secrets(s.namespace).List(metav1.ListOptions{
		LabelSelector: s.labelKV.String(),
	})
	if err != nil {
		return nil, err
	}
	var secretNames []string
	for _, scr := range secrets.Items {
		secretNames = append(secretNames, scr.Name)
	}
	return secretNames, nil
}

func (s kubernetesSecretsStore) Store(name string, data []byte) error {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				s.labelKV.getKey(): s.labelKV.getValue(),
			},
		},
		Data: map[string][]byte{
			name: data,
		},
	}
	_, err := s.Read(name)
	if err == nil {
		if _, err := s.client.Secrets(s.namespace).Update(secret); err != nil {
			return err
		}
		return nil
	}
	if _, err := s.client.Secrets(s.namespace).Create(secret); err != nil {
		return err
	}
	return nil
}

func (s kubernetesSecretsStore) Read(name string) ([]byte, error) {
	secret, err := s.client.Secrets(s.namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		if err == errors.NewNotFound(schema.GroupResource{Resource: "secrets"}, name) {
			return nil, cnabCrud.ErrRecordDoesNotExist
		}
		return nil, err
	}
	return secret.Data[name], nil
}

func (s kubernetesSecretsStore) Delete(name string) error {
	if err := s.client.Secrets(s.namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func (s kubernetesSecretsStore) ensureNamespace() error {
	namespaceService := NewNamespaceService(s.client)
	if _, err := namespaceService.Get(s.namespace); err != nil {
		_, err = namespaceService.Create(s.namespace)
		return err
	}
	return nil
}

type NamespaceService struct {
	client corev1.CoreV1Interface
}

func NewNamespaceService(clientset corev1.CoreV1Interface) *NamespaceService {
	return &NamespaceService{
		client: clientset,
	}
}

func (n NamespaceService) Get(name string) (*v1.Namespace, error) {
	namespace, err := n.client.Namespaces().Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return namespace, nil
}

func (n NamespaceService) Create(name string) (*v1.Namespace, error) {
	namespace, err := n.client.Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	})
	if err != nil {
		return nil, err
	}
	return namespace, nil
}

// FIXME Only reading from kubectl default file config for now
func getClient() (corev1.CoreV1Interface, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	kubeconfig := filepath.Join(home, ".kube", "config")
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	// create the client
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset.CoreV1(), nil
}
