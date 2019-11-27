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
	// CnabNamespace is the namespace used with the CNAB well known custom actions
	CnabNamespace = "io.cnab."

	// ActionStatusNameDeprecated is the name of the docker custom "status" action
	// Deprecated: use ActionStatusName instead
	ActionStatusNameDeprecated = Namespace + "status"
	// ActionStatusName is the name of the CNAB well known custom "status" action - TODO: Extract this constant to the cnab-go library
	ActionStatusName = CnabNamespace + "status"
	// ActionStatusJSONName is the name of the CNAB well known custom "status+json" action - TODO: Extract this constant to the cnab-go library
	ActionStatusJSONName = CnabNamespace + "status+json"
	// ActionInspectName is the name of the custom "inspect" action
	ActionInspectName = Namespace + "inspect"
	// ActionRenderName is the name of the custom "render" action
	ActionRenderName = Namespace + "render"

	// CredentialDockerContextName is the name of the credential containing a Docker context
	CredentialDockerContextName = Namespace + "context"
	// CredentialDockerContextPath is the path to the credential containing a Docker context
	CredentialDockerContextPath = "/cnab/app/context.dockercontext"
	// CredentialRegistryName is the name of the credential containing registry credentials
	CredentialRegistryName = Namespace + "registry-creds"
	// CredentialRegistryPath is the name to the credential containing registry credentials
	CredentialRegistryPath = "/cnab/app/registry-creds.json"

	// ParameterOrchestratorName is the name of the parameter containing the orchestrator
	ParameterOrchestratorName = Namespace + "orchestrator"
	// ParameterKubernetesNamespaceName is the name of the parameter containing the kubernetes namespace
	ParameterKubernetesNamespaceName = Namespace + "kubernetes-namespace"
	// ParameterRenderFormatName is the name of the parameter containing the render format
	ParameterRenderFormatName = Namespace + "render-format"
	// ParameterInspectFormatName is the name of the parameter containing the inspect format
	ParameterInspectFormatName = Namespace + "inspect-format"
	// ParameterArgs is the name of the parameter containing labels to be applied to service containers
	ParameterArgs = Namespace + "args"
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
	// DockerInspectFormatEnvVar is the environment variable set by the CNAB runtime to select
	// the inspect output format.
	DockerInspectFormatEnvVar = "DOCKER_INSPECT_FORMAT"

	DockerArgsPath = "/cnab/app/args.json"

	// CustomDockerAppName is the custom variable set by Docker App to
	// save custom informations
	CustomDockerAppName = "com.docker.app"

	// LabelAppNamespace is the label used to track app resources
	LabelAppNamespace = Namespace + "namespace"
	// LabelAppVersion is the label used to identify what version of docker app was used to create the app
	LabelAppVersion = Namespace + "version"
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
	if strings.HasSuffix(filepath.Clean(appName), AppExtension) {
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
