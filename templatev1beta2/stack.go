package templatev1beta2

import (
	"time"

	types "github.com/docker/lunchbox/templatetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// StackList is a list of stacks
type StackList struct {
	metav1.TypeMeta `yaml:",inline"`
	metav1.ListMeta `yaml:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Stack `yaml:"items" protobuf:"bytes,2,rep,name=items"`
}

// Stack is v1beta2's representation of a Stack
type Stack struct {
	metav1.TypeMeta   `yaml:",inline" yaml:",inline"`
	metav1.ObjectMeta `yaml:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   *StackSpec   `yaml:"spec,omitempty"`
	Status *StackStatus `yaml:"status,omitempty"`
}

// DeepCopyObject clones the stack
func (s *Stack) DeepCopyObject() runtime.Object {
	return s.clone()
}

// DeepCopyObject clones the stack list
func (s *StackList) DeepCopyObject() runtime.Object {
	if s == nil {
		return nil
	}
	result := new(StackList)
	result.TypeMeta = s.TypeMeta
	result.ListMeta = s.ListMeta
	if s.Items == nil {
		return result
	}
	result.Items = make([]Stack, len(s.Items))
	for ix, s := range s.Items {
		result.Items[ix] = *s.clone()
	}
	return result
}

func (s *Stack) clone() *Stack {
	if s == nil {
		return nil
	}
	result := new(Stack)
	result.TypeMeta = s.TypeMeta
	result.ObjectMeta = s.ObjectMeta
	result.Spec = s.Spec.clone()
	result.Status = s.Status.clone()
	return result
}

// StackSpec defines the desired state of Stack
type StackSpec struct {
	Services []ServiceConfig            `yaml:"services,omitempty"`
	Secrets  map[string]SecretConfig    `yaml:"secrets,omitempty"`
	Configs  map[string]ConfigObjConfig `yaml:"configs,omitempty"`
}

// ServiceConfig is the configuration of one service
type ServiceConfig struct {
	Name string `yaml:"name,omitempty"`

	CapAdd          []string                 `yaml:"cap_add,omitempty"`
	CapDrop         []string                 `yaml:"cap_drop,omitempty"`
	Command         []string                 `yaml:"command,omitempty"`
	Configs         []ServiceConfigObjConfig `yaml:"configs,omitempty"`
	Deploy          DeployConfig             `yaml:"deploy,omitempty"`
	Entrypoint      []string                 `yaml:"entrypoint,omitempty"`
	Environment     map[string]*string       `yaml:"environment,omitempty"`
	ExtraHosts      []string                 `yaml:"extra_hosts,omitempty"`
	Hostname        string                   `yaml:"hostname,omitempty"`
	HealthCheck     *HealthCheckConfig       `yaml:"health_check,omitempty"`
	Image           string                   `yaml:"image,omitempty"`
	Ipc             string                   `yaml:"ipc,omitempty"`
	Labels          map[string]string        `yaml:"labels,omitempty"`
	Pid             string                   `yaml:"pid,omitempty"`
	Ports           []ServicePortConfig      `yaml:"ports,omitempty"`
	Privileged      types.BoolOrTemplate     `yaml:"template_privileged,omitempty" yaml:"template_privileged,omitempty"`
	ReadOnly        bool                     `yaml:"read_only,omitempty"`
	Secrets         []ServiceSecretConfig    `yaml:"secrets,omitempty"`
	StdinOpen       bool                     `yaml:"stdin_open,omitempty"`
	StopGracePeriod *time.Duration           `yaml:"stop_grace_period,omitempty"`
	Tmpfs           []string                 `yaml:"tmpfs,omitempty"`
	Tty             bool                     `yaml:"tty,omitempty"`
	User            *int64                   `yaml:"user,omitempty"`
	Volumes         []ServiceVolumeConfig    `yaml:"volumes,omitempty"`
	WorkingDir      string                   `yaml:"working_dir,omitempty"`
}

// ServicePortConfig is the port configuration for a service
type ServicePortConfig struct {
	Mode      string `yaml:"mode,omitempty"`
	Target    uint32 `yaml:"target,omitempty"`
	Published uint32 `yaml:"published,omitempty"`
	Protocol  string `yaml:"protocol,omitempty"`
}

// FileObjectConfig is a config type for a file used by a service
type FileObjectConfig struct {
	Name     string            `yaml:"name,omitempty"`
	File     string            `yaml:"file,omitempty"`
	External External          `yaml:"external,omitempty"`
	Labels   map[string]string `yaml:"labels,omitempty"`
}

// SecretConfig for a secret
type SecretConfig FileObjectConfig

// ConfigObjConfig is the config for the swarm "Config" object
type ConfigObjConfig FileObjectConfig

// External identifies a Volume or Network as a reference to a resource that is
// not managed, and should already exist.
// External.name is deprecated and replaced by Volume.name
type External struct {
	Name     string `yaml:"name,omitempty"`
	External bool   `yaml:"external,omitempty"`
}

// FileReferenceConfig for a reference to a swarm file object
type FileReferenceConfig struct {
	Source string  `yaml:"source,omitempty"`
	Target string  `yaml:"target,omitempty"`
	UID    string  `yaml:"uid,omitempty"`
	GID    string  `yaml:"gid,omitempty"`
	Mode   *uint32 `yaml:"mode,omitempty"`
}

// ServiceConfigObjConfig is the config obj configuration for a service
type ServiceConfigObjConfig FileReferenceConfig

// ServiceSecretConfig is the secret configuration for a service
type ServiceSecretConfig FileReferenceConfig

// DeployConfig is the deployment configuration for a service
type DeployConfig struct {
	Mode          string            `yaml:"mode,omitempty"`
	Replicas      *uint64           `yaml:"replicas,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
	UpdateConfig  *UpdateConfig     `yaml:"update_config,omitempty"`
	Resources     Resources         `yaml:"resources,omitempty"`
	RestartPolicy *RestartPolicy    `yaml:"restart_policy,omitempty"`
	Placement     Placement         `yaml:"placement,omitempty"`
}

// UpdateConfig is the service update configuration
type UpdateConfig struct {
	Parallelism *uint64 `yaml:"paralellism,omitempty"`
}

// Resources the resource limits and reservations
type Resources struct {
	Limits       *Resource `yaml:"limits,omitempty"`
	Reservations *Resource `yaml:"reservations,omitempty"`
}

// Resource is a resource to be limited or reserved
type Resource struct {
	NanoCPUs    string `yaml:"cpus,omitempty"`
	MemoryBytes int64  `yaml:"memory,omitempty"`
}

// RestartPolicy is the service restart policy
type RestartPolicy struct {
	Condition string `yaml:"condition,omitempty"`
}

// Placement constraints for the service
type Placement struct {
	Constraints *Constraints `yaml:"constraints,omitempty"`
}

// Constraints lists constraints that can be set on the service
type Constraints struct {
	OperatingSystem *Constraint
	Architecture    *Constraint
	Hostname        *Constraint
	MatchLabels     map[string]Constraint
}

// Constraint defines a constraint and it's operator (== or !=)
type Constraint struct {
	Value    string
	Operator string
}

// HealthCheckConfig the healthcheck configuration for a service
type HealthCheckConfig struct {
	Test     []string       `yaml:"test,omitempty"`
	Timeout  *time.Duration `yaml:"timeout,omitempty"`
	Interval *time.Duration `yaml:"interval,omitempty"`
	Retries  *uint64        `yaml:"retries,omitempty"`
}

// ServiceVolumeConfig are references to a volume used by a service
type ServiceVolumeConfig struct {
	Type     string `yaml:"type,omitempty"`
	Source   string `yaml:"source,omitempty"`
	Target   string `yaml:"target,omitempty"`
	ReadOnly bool   `yaml:"read_only,omitempty"`
}

func (s *StackSpec) clone() *StackSpec {
	if s == nil {
		return nil
	}
	result := *s
	return &result
}

// StackPhase is the deployment phase of a stack
type StackPhase string

// These are valid conditions of a stack.
const (
	// StackAvailable means the stack is available.
	StackAvailable StackPhase = "Available"
	// StackProgressing means the deployment is progressing.
	StackProgressing StackPhase = "Progressing"
	// StackFailure is added in a stack when one of its members fails to be created
	// or deleted.
	StackFailure StackPhase = "Failure"
)

// StackStatus defines the observed state of Stack
type StackStatus struct {
	// Current condition of the stack.
	// +optional
	Phase StackPhase `yaml:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=StackPhase"`
	// A human readable message indicating details about the stack.
	// +optional
	Message string `yaml:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

func (s *StackStatus) clone() *StackStatus {
	if s == nil {
		return nil
	}
	result := *s
	return &result
}

// Clone clones a Stack
func (s *Stack) Clone() *Stack {
	return s.clone()
}
