package v1beta2

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// StackList is a list of stacks
type StackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Items []Stack `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// Stack is v1beta2's representation of a Stack
type Stack struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   *StackSpec   `json:"spec,omitempty"`
	Status *StackStatus `json:"status,omitempty"`
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
	Services []ServiceConfig            `json:"services,omitempty"`
	Secrets  map[string]SecretConfig    `json:"secrets,omitempty"`
	Configs  map[string]ConfigObjConfig `json:"configs,omitempty"`
}

// ServiceConfig is the configuration of one service
type ServiceConfig struct {
	Name string `json:"name,omitempty"`

	CapAdd          []string                 `json:"cap_add,omitempty" yaml:"cap_add,omitempty"`
	CapDrop         []string                 `json:"cap_drop,omitempty" yaml:"cap_drop,omitempty"`
	Command         []string                 `json:"command,omitempty"  yaml:"command,omitempty"`
	Configs         []ServiceConfigObjConfig `json:"configs,omitempty" yaml:"configs,omitempty"`
	Deploy          DeployConfig             `json:"deploy,omitempty" yaml:"deploy,omitempty"`
	Entrypoint      []string                 `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	Environment     map[string]*string       `json:"environment,omitempty" yaml:"environment,omitempty"`
	ExtraHosts      []string                 `json:"extra_hosts,omitempty" yaml:"extra_hosts,omitempty"`
	Hostname        string                   `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	HealthCheck     *HealthCheckConfig       `json:"health_check,omitempty" yaml:"health_check,omitempty"`
	Image           string                   `json:"image,omitempty" yaml:"image,omitempty"`
	Ipc             string                   `json:"ipc,omitempty" yaml:"ipc,omitempty"`
	Labels          map[string]string        `json:"labels,omitempty" yaml:"labels,omitempty"`
	Pid             string                   `json:"pid,omitempty" yaml:"pid,omitempty"`
	Ports           []ServicePortConfig      `json:"ports,omitempty" yaml:"ports,omitempty"`
	Privileged      bool                     `json:"privileged,omitempty" yaml:"privileged,omitempty"`
	ReadOnly        bool                     `json:"read_only,omitempty" yaml:"read_only,omitempty"`
	Secrets         []ServiceSecretConfig    `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	StdinOpen       bool                     `json:"stdin_open,omitempty" yaml:"stdin_open,omitempty"`
	StopGracePeriod *time.Duration           `json:"stop_grace_period,omitempty" yaml:"stop_grace_period,omitempty"`
	Tmpfs           []string                 `json:"tmpfs,omitempty" yaml:"tmpfs,omitempty"`
	Tty             bool                     `json:"tty,omitempty" yaml:"tty,omitempty"`
	User            *int64                   `json:"user,omitempty" yaml:"user,omitempty"`
	Volumes         []ServiceVolumeConfig    `json:"volumes,omitempty" yaml:"volumes,omitempty"`
	WorkingDir      string                   `json:"working_dir,omitempty" yaml:"working_dir,omitempty"`
}

// ServicePortConfig is the port configuration for a service
type ServicePortConfig struct {
	Mode      string `json:"mode,omitempty" yaml:"mode,omitempty"`
	Target    uint32 `json:"target,omitempty" yaml:"target,omitempty"`
	Published uint32 `json:"published,omitempty" yaml:"published,omitempty"`
	Protocol  string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

// FileObjectConfig is a config type for a file used by a service
type FileObjectConfig struct {
	Name     string            `json:"name,omitempty" yaml:"name,omitempty"`
	File     string            `json:"file,omitempty" yaml:"file,omitempty"`
	External External          `json:"external,omitempty" yaml:"external,omitempty"`
	Labels   map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// SecretConfig for a secret
type SecretConfig FileObjectConfig

// ConfigObjConfig is the config for the swarm "Config" object
type ConfigObjConfig FileObjectConfig

// External identifies a Volume or Network as a reference to a resource that is
// not managed, and should already exist.
// External.name is deprecated and replaced by Volume.name
type External struct {
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
	External bool   `json:"external,omitempty" yaml:"external,omitempty"`
}

// FileReferenceConfig for a reference to a swarm file object
type FileReferenceConfig struct {
	Source string  `json:"source,omitempty" yaml:"source,omitempty"`
	Target string  `json:"target,omitempty" yaml:"target,omitempty"`
	UID    string  `json:"uid,omitempty" yaml:"uid,omitempty"`
	GID    string  `json:"gid,omitempty" yaml:"gid,omitempty"`
	Mode   *uint32 `json:"mode,omitempty" yaml:"mode,omitempty"`
}

// ServiceConfigObjConfig is the config obj configuration for a service
type ServiceConfigObjConfig FileReferenceConfig

// ServiceSecretConfig is the secret configuration for a service
type ServiceSecretConfig FileReferenceConfig

// DeployConfig is the deployment configuration for a service
type DeployConfig struct {
	Mode          string            `json:"mode,omitempty" yaml:"mode,omitempty"`
	Replicas      *uint64           `json:"replicas,omitempty" yaml:"replicas,omitempty"`
	Labels        map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	UpdateConfig  *UpdateConfig     `json:"update_config,omitempty" yaml:"update_config,omitempty"`
	Resources     Resources         `json:"resources,omitempty" yaml:"resources,omitempty"`
	RestartPolicy *RestartPolicy    `json:"restart_policy,omitempty" yaml:"restart_policy,omitempty"`
	Placement     Placement         `json:"placement,omitempty" yaml:"placement,omitempty"`
}

// UpdateConfig is the service update configuration
type UpdateConfig struct {
	Parallelism *uint64 `json:"paralellism,omitempty" yaml:"paralellism,omitempty"`
}

// Resources the resource limits and reservations
type Resources struct {
	Limits       *Resource `json:"limits,omitempty" yaml:"limits,omitempty"`
	Reservations *Resource `json:"reservations,omitempty" yaml:"reservations,omitempty"`
}

// Resource is a resource to be limited or reserved
type Resource struct {
	NanoCPUs    string `json:"cpus,omitempty" yaml:"cpus,omitempty"`
	MemoryBytes int64  `json:"memory,omitempty" yaml:"memory,omitempty"`
}

// RestartPolicy is the service restart policy
type RestartPolicy struct {
	Condition string `json:"condition,omitempty" yaml:"condition,omitempty"`
}

// Placement constraints for the service
type Placement struct {
	Constraints *Constraints `json:"constraints,omitempty" yaml:"constraints,omitempty"`
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
	Test     []string       `json:"test,omitempty" yaml:"test,omitempty"`
	Timeout  *time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Interval *time.Duration `json:"interval,omitempty" yaml:"interval,omitempty"`
	Retries  *uint64        `json:"retries,omitempty" yaml:"retries,omitempty"`
}

// ServiceVolumeConfig are references to a volume used by a service
type ServiceVolumeConfig struct {
	Type     string `json:"type,omitempty" yaml:"type,omitempty"`
	Source   string `json:"source,omitempty" yaml:"source,omitempty"`
	Target   string `json:"target,omitempty" yaml:"target,omitempty"`
	ReadOnly bool   `json:"read_only,omitempty" yaml:"read_only,omitempty"`
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
	Phase StackPhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=StackPhase"`
	// A human readable message indicating details about the stack.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
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
