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
	CapAdd          []string                               `mapstructure:"cap_add" yaml:"cap_add,omitempty"`
	CapDrop         []string                               `mapstructure:"cap_drop" yaml:"cap_drop,omitempty"`
	CgroupParent    string                                 `mapstructure:"cgroup_parent" yaml:"cgroup_parent,omitempty"`
	Command         types.ShellCommand                     `yaml:",omitempty"`
	Configs         []ServiceConfigObjConfig               `yaml:",omitempty"`
	ContainerName   string                                 `mapstructure:"container_name" yaml:"container_name,omitempty"`
	CredentialSpec  types.CredentialSpecConfig             `mapstructure:"credential_spec" yaml:"credential_spec,omitempty"`
	DependsOn       []string                               `mapstructure:"depends_on" yaml:"depends_on,omitempty"`
	Deploy          DeployConfig                           `yaml:",omitempty"`
	Devices         []string                               `yaml:",omitempty"`
	DNS             types.StringList                       `yaml:",omitempty"`
	DNSSearch       types.StringList                       `mapstructure:"dns_search" yaml:"dns_search,omitempty"`
	DomainName      string                                 `mapstructure:"domainname" yaml:"domainname,omitempty"`
	Entrypoint      types.ShellCommand                     `yaml:",omitempty"`
	Environment     types.MappingWithEquals                `yaml:",omitempty"`
	EnvFile         types.StringList                       `mapstructure:"env_file" yaml:"env_file,omitempty"`
	Expose          types.StringOrNumberList               `yaml:",omitempty"`
	ExternalLinks   []string                               `mapstructure:"external_links" yaml:"external_links,omitempty"`
	ExtraHosts      types.HostsList                        `mapstructure:"extra_hosts" yaml:"extra_hosts,omitempty"`
	Hostname        string                                 `yaml:",omitempty"`
	HealthCheck     *HealthCheckConfig                     `yaml:",omitempty"`
	Image           string                                 `yaml:",omitempty"`
	Init            *BoolOrTemplate                        `yaml:"template_init,omitempty"`
	Ipc             string                                 `yaml:",omitempty"`
	Isolation       string                                 `mapstructure:"isolation" yaml:"isolation,omitempty"`
	Labels          types.Labels                           `yaml:",omitempty"`
	Links           []string                               `yaml:",omitempty"`
	Logging         *types.LoggingConfig                   `yaml:",omitempty"`
	MacAddress      string                                 `mapstructure:"mac_address" yaml:"mac_address,omitempty"`
	NetworkMode     string                                 `mapstructure:"network_mode" yaml:"network_mode,omitempty"`
	Networks        map[string]*types.ServiceNetworkConfig `yaml:",omitempty"`
	Pid             string                                 `yaml:",omitempty"`
	Ports           []ServicePortConfig                    `yaml:",omitempty"`
	Privileged      BoolOrTemplate                         `yaml:"template_privileged,omitempty"`
	ReadOnly        BoolOrTemplate                         `mapstructure:"read_only" yaml:"template_read_only,omitempty"`
	Restart         string                                 `yaml:",omitempty"`
	Secrets         []ServiceSecretConfig                  `yaml:",omitempty"`
	SecurityOpt     []string                               `mapstructure:"security_opt" yaml:"security_opt,omitempty"`
	StdinOpen       BoolOrTemplate                         `mapstructure:"stdin_open" yaml:"template_stdin_open,omitempty"`
	StopGracePeriod DurationOrTemplate                     `mapstructure:"stop_grace_period" yaml:"template_stop_grace_period,omitempty"`
	StopSignal      string                                 `mapstructure:"stop_signal" yaml:"stop_signal,omitempty"`
	Sysctls         types.StringList                       `yaml:",omitempty"`
	Tmpfs           types.StringList                       `yaml:",omitempty"`
	Tty             BoolOrTemplate                         `mapstructure:"tty" yaml:"template_tty,omitempty"`
	Ulimits         map[string]*types.UlimitsConfig        `yaml:",omitempty"`
	User            string                                 `yaml:",omitempty"`
	UserNSMode      string                                 `mapstructure:"userns_mode" yaml:"userns_mode,omitempty"`
	Volumes         []ServiceVolumeConfig                  `yaml:",omitempty"`
	WorkingDir      string                                 `mapstructure:"working_dir" yaml:"working_dir,omitempty"`

	Extras map[string]interface{} `yaml:",inline"`
}

// DeployConfig the deployment configuration for a service
type DeployConfig struct {
	Mode           string               `yaml:",omitempty"`
	Replicas       UInt64OrTemplate     `yaml:"template_replicas,omitempty"`
	Labels         types.Labels         `yaml:",omitempty"`
	UpdateConfig   *UpdateConfig        `mapstructure:"update_config" yaml:"update_config,omitempty"`
	RollbackConfig *UpdateConfig        `mapstructure:"rollback_config" yaml:"rollback_config,omitempty"`
	Resources      Resources            `yaml:",omitempty"`
	RestartPolicy  *types.RestartPolicy `mapstructure:"restart_policy" yaml:"restart_policy,omitempty"`
	Placement      types.Placement      `yaml:",omitempty"`
	EndpointMode   string               `mapstructure:"endpoint_mode" yaml:"endpoint_mode,omitempty"`
}

// HealthCheckConfig the healthcheck configuration for a service
type HealthCheckConfig struct {
	Test        types.HealthCheckTest `yaml:",omitempty"`
	Timeout     DurationOrTemplate    `yaml:"template_timeout,omitempty"`
	Interval    DurationOrTemplate    `yaml:"template_interval,omitempty"`
	Retries     UInt64OrTemplate      `yaml:"template_retries,omitempty"`
	StartPeriod *time.Duration        `mapstructure:"start_period" yaml:"start_period,omitempty"`
	Disable     bool                  `yaml:",omitempty"`
}

// UpdateConfig the service update configuration
type UpdateConfig struct {
	Parallelism     UInt64OrTemplate `yaml:"template_parallelism,omitempty"`
	Delay           time.Duration    `yaml:",omitempty"`
	FailureAction   string           `mapstructure:"failure_action" yaml:"failure_action,omitempty"`
	Monitor         time.Duration    `yaml:",omitempty"`
	MaxFailureRatio float32          `mapstructure:"max_failure_ratio" yaml:"max_failure_ratio,omitempty"`
	Order           string           `yaml:",omitempty"`
}

// Resources the resource limits and reservations
type Resources struct {
	Limits       *Resource `yaml:",omitempty"`
	Reservations *Resource `yaml:",omitempty"`
}

// Resource is a resource to be limited or reserved
type Resource struct {
	// TODO: types to convert from units and ratios
	NanoCPUs         string                  `mapstructure:"cpus" yaml:"cpus,omitempty"`
	MemoryBytes      UnitBytesOrTemplate     `mapstructure:"memory" yaml:"template_memory,omitempty"`
	GenericResources []types.GenericResource `mapstructure:"generic_resources" yaml:"generic_resources,omitempty"`
}

// ServicePortConfig is the port configuration for a service
type ServicePortConfig struct {
	Mode      string           `yaml:",omitempty"`
	Target    UInt64OrTemplate `yaml:"template_target,omitempty"`
	Published UInt64OrTemplate `yaml:"template_published,omitempty"`
	Protocol  string           `yaml:",omitempty"`
}

// ServiceVolumeConfig are references to a volume used by a service
type ServiceVolumeConfig struct {
	Type        string                     `yaml:",omitempty"`
	Source      string                     `yaml:",omitempty"`
	Target      string                     `yaml:",omitempty"`
	ReadOnly    BoolOrTemplate             `mapstructure:"read_only" yaml:"template_read_only,omitempty"`
	Consistency string                     `yaml:",omitempty"`
	Bind        *types.ServiceVolumeBind   `yaml:",omitempty"`
	Volume      *types.ServiceVolumeVolume `yaml:",omitempty"`
	Tmpfs       *types.ServiceVolumeTmpfs  `yaml:",omitempty"`
}

// FileReferenceConfig for a reference to a swarm file object
type FileReferenceConfig struct {
	Source string           `yaml:",omitempty"`
	Target string           `yaml:",omitempty"`
	UID    string           `yaml:",omitempty"`
	GID    string           `yaml:",omitempty"`
	Mode   UInt64OrTemplate `yaml:"template_mode,omitempty"`
}

// ServiceConfigObjConfig is the config obj configuration for a service
type ServiceConfigObjConfig FileReferenceConfig

// ServiceSecretConfig is the secret configuration for a service
type ServiceSecretConfig FileReferenceConfig
