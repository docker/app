package internal

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// AppExtension is the extension used by an application.
	AppExtension = ".dockerapp"
	// ImageLabel is the label used to distinguish applications from Docker images.
	ImageLabel = "com.docker.application"
	// MetadataFileName is metadata file name
	MetadataFileName = "metadata.yml"
	// ComposeFileName is compose file name
	ComposeFileName = "docker-compose.yml"
	// ParametersFileName is parameters file name
	ParametersFileName = "parameters.yml"

	// DeprecatedSettingsFileName is the deprecated settings file name (replaced by ParametersFileName)
	DeprecatedSettingsFileName = "settings.yml"

	// Namespace is the reverse DNS namespace used with labels and CNAB custom actions.
	Namespace = "com.docker.app."

	// ActionStatusName is the name of the custom "status" action
	ActionStatusName = Namespace + "status"
	// ActionInspectName is the name of the custom "inspect" action
	ActionInspectName = Namespace + "inspect"
	// ActionRenderName is the name of the custom "render" action
	ActionRenderName = Namespace + "render"

	// CredentialDockerContextName is the name of the credential containing a Docker context
	CredentialDockerContextName = "docker.context"
	// CredentialDockerContextPath is the path to the credential containing a Docker context
	CredentialDockerContextPath = "/cnab/app/context.dockercontext"
	// CredentialRegistryName is the name of the credential containing registry credentials
	CredentialRegistryName = Namespace + "registry-creds"
	// CredentialRegistryPath is the name to the credential containing registry credentials
	CredentialRegistryPath = "/cnab/app/registry-creds.json"
	// ComposeOverridesDir is the path where automatic parameters store their value overrides
	ComposeOverridesDir = "/cnab/app/overrides"

	// ParameterOrchestratorName is the name of the parameter containing the orchestrator
	ParameterOrchestratorName = Namespace + "orchestrator"
	// ParameterKubernetesNamespaceName is the name of the parameter containing the kubernetes namespace
	ParameterKubernetesNamespaceName = Namespace + "kubernetes-namespace"
	// ParameterRenderFormatName is the name of the parameter containing the kubernetes namespace
	ParameterRenderFormatName = Namespace + "render-format"
	// ParameterShareRegistryCredsName is the name of the parameter which indicates if credentials should be shared
	ParameterShareRegistryCredsName = Namespace + "share-registry-creds"

	// DockerStackOrchestratorEnvVar is the environment variable set by the CNAB runtime to select
	// the stack orchestrator.
	DockerStackOrchestratorEnvVar = "DOCKER_STACK_ORCHESTRATOR"
	// DockerKubernetesNamespaceEnvVar is the environment variable set by the CNAB runtime to select
	// the kubernetes namespace.
	DockerKubernetesNamespaceEnvVar = "DOCKER_KUBERNETES_NAMESPACE"
	// DockerRenderFormatEnvVar is the environment variable set by the CNAB runtime to select
	// the render output format.
	DockerRenderFormatEnvVar = "DOCKER_RENDER_FORMAT"
)

var (
	// FileNames lists the application file names, in order.
	FileNames = []string{MetadataFileName, ComposeFileName, ParametersFileName}
)

var appNameRe, _ = regexp.Compile("^[a-zA-Z][a-zA-Z0-9_-]+$")

// AppNameFromDir takes a path to an app directory and returns
// the application's name
func AppNameFromDir(dirName string) string {
	return strings.TrimSuffix(filepath.Base(dirName), AppExtension)
}

// DirNameFromAppName takes an application name and returns the
// corresponding directory name
func DirNameFromAppName(appName string) string {
	if strings.HasSuffix(strings.TrimSuffix(appName, "/"), AppExtension) {
		return appName
	}
	return appName + AppExtension
}

// ValidateAppName takes an app name and returns an error if it doesn't
// match the expected format
func ValidateAppName(appName string) error {
	if appNameRe.MatchString(appName) {
		return nil
	}
	return fmt.Errorf(
		"invalid app name: %s ; app names must start with a letter, and must contain only letters, numbers, '-' and '_' (regexp: %q)",
		appName,
		appNameRe.String(),
	)
}
