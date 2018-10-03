package templatetypes

import (
	"time"

	"github.com/docker/cli/cli/compose/types"
)

// BoolOrTemplate stores a boolean or a templated string
type BoolOrTemplate struct {
	Value         bool   `yaml:",omitempty"`
	ValueTemplate string `yaml:",omitempty"`
}

// UInt64OrTemplate stores an uint64 or a templated string
type UInt64OrTemplate struct {
	Value         *uint64 `yaml:",omitempty"`
	ValueTemplate string  `yaml:",omitempty"`
}

// UnitBytesOrTemplate stores an int64 parsed from a size or a templated string
type UnitBytesOrTemplate struct {
	Value         int64  `yaml:",omitempty"`
	ValueTemplate string `yaml:",omitempty"`
}

// DurationOrTemplate stores a duration or a templated string
type DurationOrTemplate struct {
	Value         *time.Duration `yaml:",omitempty"`
	ValueTemplate string         `yaml:",omitempty"`
}

// StringTemplate contains a string that can be a template value
type StringTemplate struct {
	Value string
}

// StringTemplateList is a list of StringTemplate
type StringTemplateList []StringTemplate

// ShellCommandTemplate is a shell command parsed as a list or string
type ShellCommandTemplate []StringTemplate

// HostsListTemplate is a list to hosts parsed as a list or map
type HostsListTemplate []StringTemplate

// LabelsTemplate is a mapping type for labels
type LabelsTemplate map[StringTemplate]StringTemplate

// MappingWithEqualsTemplate is a mapping type parsed as a list or map
type MappingWithEqualsTemplate map[StringTemplate]*StringTemplate

// ProcessTemplate can be overridden to process template values when marshaling
var ProcessTemplate = func(s string) (string, error) {
	return s, nil
}

// UnmarshalYAML implements the Unmarshaler interface
func (s *StringTemplate) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(&s.Value)
}

// MarshalYAML implements the Marshaler interface
func (s StringTemplate) MarshalYAML() (interface{}, error) {
	return ProcessTemplate(s.Value)
}

// MarshalYAML implements the Marshaler interface
func (s BoolOrTemplate) MarshalYAML() (interface{}, error) {
	if s.ValueTemplate != "" {
		return ProcessTemplate(s.ValueTemplate)
	}
	return s.Value, nil
}

// MarshalYAML implements the Marshaler interface
func (s UInt64OrTemplate) MarshalYAML() (interface{}, error) {
	if s.ValueTemplate != "" {
		return ProcessTemplate(s.ValueTemplate)
	}
	return s.Value, nil
}

// MarshalYAML implements the Marshaler interface
func (s UnitBytesOrTemplate) MarshalYAML() (interface{}, error) {
	if s.ValueTemplate != "" {
		return ProcessTemplate(s.ValueTemplate)
	}
	return s.Value, nil
}

// MarshalYAML implements the Marshaler interface
func (s DurationOrTemplate) MarshalYAML() (interface{}, error) {
	if s.ValueTemplate != "" {
		return ProcessTemplate(s.ValueTemplate)
	}
	return s.Value, nil
}

// Config is a full compose file configuration
type Config struct {
	Filename string `yaml:"-"`
	Version  string
	Services Services
	Networks map[string]types.NetworkConfig   `yaml:",omitempty"`
	Volumes  map[string]types.VolumeConfig    `yaml:",omitempty"`
	Secrets  map[string]types.SecretConfig    `yaml:",omitempty"`
	Configs  map[string]types.ConfigObjConfig `yaml:",omitempty"`
}

// Services is a list of ServiceConfig
type Services []ServiceConfig

// MarshalYAML makes Services implement yaml.Marshaller
func (s Services) MarshalYAML() (interface{}, error) {
	services := map[string]ServiceConfig{}
	for _, service := range s {
		services[service.Name] = service
	}
	return services, nil
}

// ServiceConfig is the configuration of one service
type ServiceConfig struct {
	Name string `yaml:"-"`

	Build           types.BuildConfig                      `yaml:",omitempty"`
	CapAdd          []StringTemplate                       `mapstructure:"cap_add" yaml:"cap_add,omitempty"`
	CapDrop         []StringTemplate                       `mapstructure:"cap_drop" yaml:"cap_drop,omitempty"`
	CgroupParent    StringTemplate                         `mapstructure:"cgroup_parent" yaml:"cgroup_parent,omitempty"`
	Command         ShellCommandTemplate                   `yaml:",omitempty"`
	Configs         []ServiceConfigObjConfig               `yaml:",omitempty"`
	ContainerName   StringTemplate                         `mapstructure:"container_name" yaml:"container_name,omitempty"`
	CredentialSpec  types.CredentialSpecConfig             `mapstructure:"credential_spec" yaml:"credential_spec,omitempty"`
	DependsOn       []StringTemplate                       `mapstructure:"depends_on" yaml:"depends_on,omitempty"`
	Deploy          DeployConfig                           `yaml:",omitempty"`
	Devices         []StringTemplate                       `yaml:",omitempty"`
	DNS             StringTemplateList                     `yaml:",omitempty"`
	DNSSearch       StringTemplateList                     `mapstructure:"dns_search" yaml:"dns_search,omitempty"`
	DomainName      StringTemplate                         `mapstructure:"domainname" yaml:"domainname,omitempty"`
	Entrypoint      ShellCommandTemplate                   `yaml:",omitempty"`
	Environment     MappingWithEqualsTemplate              `yaml:",omitempty"`
	EnvFile         StringTemplateList                     `mapstructure:"env_file" yaml:"env_file,omitempty"`
	Expose          StringTemplateList                     `yaml:",omitempty"`
	ExternalLinks   []StringTemplate                       `mapstructure:"external_links" yaml:"external_links,omitempty"`
	ExtraHosts      HostsListTemplate                      `mapstructure:"extra_hosts" yaml:"extra_hosts,omitempty"`
	Hostname        StringTemplate                         `yaml:",omitempty"`
	HealthCheck     *HealthCheckConfig                     `yaml:",omitempty"`
	Image           StringTemplate                         `yaml:",omitempty"`
	Init            *BoolOrTemplate                        `yaml:"init,omitempty"`
	Ipc             StringTemplate                         `yaml:",omitempty"`
	Isolation       StringTemplate                         `mapstructure:"isolation" yaml:"isolation,omitempty"`
	Labels          LabelsTemplate                         `yaml:",omitempty"`
	Links           []StringTemplate                       `yaml:",omitempty"`
	Logging         *types.LoggingConfig                   `yaml:",omitempty"`
	MacAddress      StringTemplate                         `mapstructure:"mac_address" yaml:"mac_address,omitempty"`
	NetworkMode     StringTemplate                         `mapstructure:"network_mode" yaml:"network_mode,omitempty"`
	Networks        map[string]*types.ServiceNetworkConfig `yaml:",omitempty"`
	Pid             StringTemplate                         `yaml:",omitempty"`
	Ports           []ServicePortConfig                    `yaml:",omitempty"`
	Privileged      BoolOrTemplate                         `yaml:"privileged,omitempty"`
	ReadOnly        BoolOrTemplate                         `mapstructure:"read_only" yaml:"read_only,omitempty"`
	Restart         StringTemplate                         `yaml:",omitempty"`
	Secrets         []ServiceSecretConfig                  `yaml:",omitempty"`
	SecurityOpt     []StringTemplate                       `mapstructure:"security_opt" yaml:"security_opt,omitempty"`
	StdinOpen       BoolOrTemplate                         `mapstructure:"stdin_open" yaml:"stdin_open,omitempty"`
	StopGracePeriod DurationOrTemplate                     `mapstructure:"stop_grace_period" yaml:"stop_grace_period,omitempty"`
	StopSignal      StringTemplate                         `mapstructure:"stop_signal" yaml:"stop_signal,omitempty"`
	Sysctls         StringTemplateList                     `yaml:",omitempty"`
	Tmpfs           StringTemplateList                     `yaml:",omitempty"`
	Tty             BoolOrTemplate                         `mapstructure:"tty" yaml:"tty,omitempty"`
	Ulimits         map[string]*types.UlimitsConfig        `yaml:",omitempty"`
	User            StringTemplate                         `yaml:",omitempty"`
	UserNSMode      StringTemplate                         `mapstructure:"userns_mode" yaml:"userns_mode,omitempty"`
	Volumes         []ServiceVolumeConfig                  `yaml:",omitempty"`
	WorkingDir      StringTemplate                         `mapstructure:"working_dir" yaml:"working_dir,omitempty"`

	Extras map[string]interface{} `yaml:",inline"`
}

// DeployConfig the deployment configuration for a service
type DeployConfig struct {
	Mode           StringTemplate       `yaml:",omitempty"`
	Replicas       UInt64OrTemplate     `yaml:"replicas,omitempty"`
	Labels         LabelsTemplate       `yaml:",omitempty"`
	UpdateConfig   *UpdateConfig        `mapstructure:"update_config" yaml:"update_config,omitempty"`
	RollbackConfig *UpdateConfig        `mapstructure:"rollback_config" yaml:"rollback_config,omitempty"`
	Resources      Resources            `yaml:",omitempty"`
	RestartPolicy  *types.RestartPolicy `mapstructure:"restart_policy" yaml:"restart_policy,omitempty"`
	Placement      types.Placement      `yaml:",omitempty"`
	EndpointMode   StringTemplate       `mapstructure:"endpoint_mode" yaml:"endpoint_mode,omitempty"`
}

// HealthCheckConfig the healthcheck configuration for a service
type HealthCheckConfig struct {
	Test        types.HealthCheckTest `yaml:",omitempty"`
	Timeout     DurationOrTemplate    `yaml:"timeout,omitempty"`
	Interval    DurationOrTemplate    `yaml:"interval,omitempty"`
	Retries     UInt64OrTemplate      `yaml:"retries,omitempty"`
	StartPeriod *time.Duration        `mapstructure:"start_period" yaml:"start_period,omitempty"`
	Disable     BoolOrTemplate        `yaml:",omitempty"`
}

// UpdateConfig the service update configuration
type UpdateConfig struct {
	Parallelism     UInt64OrTemplate `yaml:"parallelism,omitempty"`
	Delay           time.Duration    `yaml:",omitempty"`
	FailureAction   StringTemplate   `mapstructure:"failure_action" yaml:"failure_action,omitempty"`
	Monitor         time.Duration    `yaml:",omitempty"`
	MaxFailureRatio float32          `mapstructure:"max_failure_ratio" yaml:"max_failure_ratio,omitempty"`
	Order           StringTemplate   `yaml:",omitempty"`
}

// Resources the resource limits and reservations
type Resources struct {
	Limits       *Resource `yaml:",omitempty"`
	Reservations *Resource `yaml:",omitempty"`
}

// Resource is a resource to be limited or reserved
type Resource struct {
	// TODO: types to convert from units and ratios
	NanoCPUs         StringTemplate          `mapstructure:"cpus" yaml:"cpus,omitempty"`
	MemoryBytes      UnitBytesOrTemplate     `mapstructure:"memory" yaml:"memory,omitempty"`
	GenericResources []types.GenericResource `mapstructure:"generic_resources" yaml:"generic_resources,omitempty"`
}

// ServicePortConfig is the port configuration for a service
type ServicePortConfig struct {
	Mode      StringTemplate   `yaml:",omitempty"`
	Target    UInt64OrTemplate `yaml:"target,omitempty"`
	Published UInt64OrTemplate `yaml:"published,omitempty"`
	Protocol  StringTemplate   `yaml:",omitempty"`
}

// ServiceVolumeConfig are references to a volume used by a service
type ServiceVolumeConfig struct {
	Type        string                     `yaml:",omitempty"`
	Source      StringTemplate             `yaml:",omitempty"`
	Target      StringTemplate             `yaml:",omitempty"`
	ReadOnly    BoolOrTemplate             `mapstructure:"read_only" yaml:"read_only,omitempty"`
	Consistency StringTemplate             `yaml:",omitempty"`
	Bind        *types.ServiceVolumeBind   `yaml:",omitempty"`
	Volume      *types.ServiceVolumeVolume `yaml:",omitempty"`
	Tmpfs       *types.ServiceVolumeTmpfs  `yaml:",omitempty"`
}

// FileReferenceConfig for a reference to a swarm file object
type FileReferenceConfig struct {
	Source StringTemplate   `yaml:",omitempty"`
	Target StringTemplate   `yaml:",omitempty"`
	UID    StringTemplate   `yaml:",omitempty"`
	GID    StringTemplate   `yaml:",omitempty"`
	Mode   UInt64OrTemplate `yaml:"mode,omitempty"`
}

// ServiceConfigObjConfig is the config obj configuration for a service
type ServiceConfigObjConfig FileReferenceConfig

// ServiceSecretConfig is the secret configuration for a service
type ServiceSecretConfig FileReferenceConfig
