package model

import (
	"fmt"
	"github.com/gigurra/flycd/pkg/util/util_math"
	"github.com/samber/lo"
	"os"
	"regexp"
)

type Concurrency struct {
	Type      string `yaml:"type" toml:"type"`
	SoftLimit int    `yaml:"soft_limit" toml:"soft_limit"`
	HardLimit int    `yaml:"hard_limit" toml:"hard_limit"`
}

type Port struct {
	Handlers   []string `yaml:"handlers" toml:"handlers"`
	Port       int      `yaml:"port" toml:"port"`
	ForceHttps bool     `yaml:"force_https" toml:"force_https"`
}

type Service struct {
	InternalPort       int         `yaml:"internal_port" toml:"internal_port"`
	Protocol           string      `yaml:"protocol" toml:"protocol"`
	ForceHttps         bool        `yaml:"force_https" toml:"force_https"`
	AutoStopMachines   bool        `yaml:"auto_stop_machines" toml:"auto_stop_machines"`
	AutoStartMachines  bool        `yaml:"auto_start_machines" toml:"auto_start_machines"`
	MinMachinesRunning int         `yaml:"min_machines_running" toml:"min_machines_running"`
	Concurrency        Concurrency `yaml:"concurrency" toml:"concurrency,omitempty"`
	Ports              []Port      `yaml:"ports" toml:"ports,omitempty"`
	Processes          []string    `yaml:"processes" toml:"processes,omitempty"`
}

type HttpService struct {
	InternalPort       int         `yaml:"internal_port" toml:"internal_port"`
	ForceHttps         bool        `yaml:"force_https" toml:"force_https"`
	AutoStopMachines   bool        `yaml:"auto_stop_machines" toml:"auto_stop_machines"`
	AutoStartMachines  bool        `yaml:"auto_start_machines" toml:"auto_start_machines"`
	MinMachinesRunning int         `yaml:"min_machines_running" toml:"min_machines_running"`
	Concurrency        Concurrency `yaml:"concurrency" toml:"concurrency,omitempty"`
	Processes          []string    `yaml:"processes" toml:"processes,omitempty"`
}

func (s HttpService) IsEmpty() bool {
	return s.InternalPort == 0 &&
		s.ForceHttps == false &&
		s.AutoStopMachines == false &&
		s.AutoStartMachines == false &&
		s.MinMachinesRunning == 0 &&
		s.Concurrency == Concurrency{} &&
		len(s.Processes) == 0
}

type Mount struct {
	Source      string `yaml:"source" toml:"source"`
	Destination string `yaml:"destination" toml:"destination"`
}

type PreCalculatedAppConfig struct {
	Typed   AppConfig
	UnTyped map[string]any
}

type MachineConfig struct {
	Count          int            `yaml:"count" toml:"count"` // default
	CountPerRegion map[string]int `yaml:"count_per_region" toml:"count_per_region"`
}

func (m MachineConfig) CountInRegion(region string) int {
	count, ok := m.CountPerRegion[region]
	if !ok {
		return m.Count
	}
	return count
}

type AppConfig struct {
	App           string            `yaml:"app" toml:"app"`
	Org           string            `yaml:"org" toml:"org,omitempty"`
	PrimaryRegion string            `yaml:"primary_region" toml:"primary_region,omitempty"`
	ExtraRegions  []string          `yaml:"extra_regions,omitempty" toml:"extra_regions,omitempty"`
	Source        Source            `yaml:"source,omitempty" toml:"source"`
	MergeCfg      MergeCfg          `yaml:"merge_cfg,omitempty" toml:"merge_cfg" json:"merge_cfg,omitempty"`
	Services      []Service         `yaml:"services,omitempty" toml:"services,omitempty"`
	HttpService   *HttpService      `yaml:"http_service,omitempty" toml:"http_service,omitempty"`
	LaunchParams  []string          `yaml:"launch_params,omitempty" toml:"launch_params,omitempty"`
	DeployParams  []string          `yaml:"deploy_params,omitempty" toml:"deploy_params"`
	Env           map[string]string `yaml:"env,omitempty" toml:"env,omitempty"`
	Build         map[string]any    `yaml:"build,omitempty" toml:"build,omitempty"`
	Mounts        []Mount           `yaml:"mounts,omitempty" toml:"mounts,omitempty"` // fly.io only supports one mount :S
	Volumes       []VolumeConfig    `yaml:"volumes,omitempty" toml:"volumes,omitempty"`
	Machines      MachineConfig     `yaml:"machines,omitempty" toml:"machines,omitempty"`
	Secrets       []SecretRef       `yaml:"secrets,omitempty" toml:"secrets,omitempty"`
	NetworkConfig NetworkConfig     `yaml:"network,omitempty" toml:"network,omitempty"`
}

func (a *AppConfig) RegionsWPrimaryLast() []string {
	result := []string{}
	result = append(result, a.ExtraRegions...)
	result = append(result, a.PrimaryRegion) // last to ensure we deploy it last (saves hash values)
	return lo.Uniq(result)
}

func (a *AppConfig) MinMachinesFromServices() int {
	return util_math.Max(
		func() int {
			if a.HttpService != nil {
				return a.HttpService.MinMachinesRunning
			} else {
				return 0
			}
		}(),
		lo.Reduce(
			a.Services,
			func(agg int, item Service, _ int) int { return util_math.Max(agg, item.MinMachinesRunning) },
			0,
		),
	)
}

type SecretSourceType string

const (
	SecretSourceTypeEnv SecretSourceType = "env"
	SecretSourceTypeRaw SecretSourceType = "raw" // not recommended
	// Add more when needed
)

type SecretRef struct {
	Name string           `yaml:"name" toml:"name"`
	Type SecretSourceType `yaml:"type" toml:"type"`
	Env  string           `yaml:"env" toml:"env"`
	Raw  string           `yaml:"raw" toml:"raw"`
}

func (s SecretRef) GetSecretValue() (string, error) {
	switch s.Type {
	case SecretSourceTypeEnv:
		if s.Env == "" {
			value, exists := os.LookupEnv(s.Name)
			if !exists {
				return "", fmt.Errorf("env var %s for secret %s does not exist", s.Name, s.Name)
			} else {
				return value, nil
			}
		} else {
			value, exists := os.LookupEnv(s.Env)
			if !exists {
				return "", fmt.Errorf("env var %s for secret %s does not exist", s.Env, s.Name)
			} else {
				return value, nil
			}
		}
	case SecretSourceTypeRaw:
		return s.Raw, nil
	default:
		return "", fmt.Errorf("unknown secret type: %s", s.Type)
	}
}

type NetworkConfig struct {
	Ips          []IpConfig `yaml:"ips" toml:"ips"`
	AutoPruneIps bool       `yaml:"auto_prune_ips" toml:"auto_prune_ips"`
}

func (c NetworkConfig) Validate() error {
	for _, ip := range c.Ips {
		err := ip.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c NetworkConfig) IsEmpty() bool {
	return len(c.Ips) == 0 && !c.AutoPruneIps
}

type Ipv string

const (
	IpV4   Ipv = "v4"
	IpV6   Ipv = "v6"
	IpVUkn Ipv = "unknown"
)

type IpConfig struct {
	V       Ipv    `yaml:"v" toml:"v"`
	Network string `yaml:"network" toml:"network"`
	Org     string `yaml:"org" toml:"org"`
	Private bool   `yaml:"private" toml:"private"`
	Shared  bool   `yaml:"shared" toml:"shared"`
	Region  string `yaml:"region" toml:"region"`
}

func (c IpConfig) Validate() error {
	if c.V == "" {
		return fmt.Errorf("ip config missing v")
	}
	if c.V != IpV4 && c.V != IpV6 {
		return fmt.Errorf("ip config v must be either v4 or v6")
	}

	if c.V == IpV4 && c.Private {
		return fmt.Errorf("ip config v4 cannot be private")
	}

	if c.V == IpV4 && c.Network != "" {
		return fmt.Errorf("ip config v4 cannot have network")
	}

	if c.V == IpV4 && c.Org != "" {
		return fmt.Errorf("ip config v4 cannot have org")
	}

	if c.V == IpV4 && c.Region != "" {
		return fmt.Errorf("ip config v4 cannot have region")
	}

	if c.V == IpV6 && c.Shared {
		return fmt.Errorf("ip config v6 cannot be shared")
	}
	return nil
}

type ValidateAppConfigOptions struct {
	ValidateSource bool
}

func (opts ValidateAppConfigOptions) WithValidateSource(validateSource ...bool) ValidateAppConfigOptions {
	if len(validateSource) > 0 {
		opts.ValidateSource = validateSource[0]
	} else {
		opts.ValidateSource = true
	}
	return opts
}

func NewValidateAppConfigOptions() ValidateAppConfigOptions {
	return ValidateAppConfigOptions{
		ValidateSource: true,
	}
}

func (a *AppConfig) Validate(options ...ValidateAppConfigOptions) error {
	if a.App == "" {
		return fmt.Errorf("app name is required")
	}

	if a.PrimaryRegion == "" {
		return fmt.Errorf("primary_region is required")
	}

	opts := NewValidateAppConfigOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	if a.Env == nil {
		a.Env = make(map[string]string)
	}

	if a.Build == nil {
		a.Build = make(map[string]any)
	}

	if a.Services == nil {
		a.Services = []Service{}
	}

	if a.LaunchParams == nil {
		a.LaunchParams = []string{}
	}

	if a.DeployParams == nil {
		a.DeployParams = []string{}
	}

	if a.Mounts == nil {
		a.Mounts = []Mount{}
	}

	if a.MergeCfg.Include == nil {
		a.MergeCfg.Include = []string{}
	}

	if a.Volumes == nil {
		a.Volumes = []VolumeConfig{}
	}

	if a.ExtraRegions == nil {
		a.ExtraRegions = []string{}
	}

	err := a.NetworkConfig.Validate()
	if err != nil {
		return fmt.Errorf("network config validation failed: %w", err)
	}

	// only permit apps that are valid dns names
	const subdomainPrefixRegExp = `^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`

	subdomainPrefixChecker := regexp.MustCompile(subdomainPrefixRegExp)
	if !subdomainPrefixChecker.MatchString(a.App) {
		return fmt.Errorf("app name '%s' is not a valid subdomain prefix", a.App)
	}

	if opts.ValidateSource {
		err := a.Source.Validate()
		if err != nil {
			return err
		}
	}

	if len(a.LaunchParams) == 0 {
		a.LaunchParams = NewDefaultLaunchParams(a.App, a.Org)
	}

	if a.Org != "" && a.PrimaryRegion == "" {
		return fmt.Errorf("primary_region is required when org is specified")
	}

	return nil
}

func NewDefaultServiceConfig() Service {
	return Service{
		InternalPort:       80,
		Protocol:           "tcp",
		ForceHttps:         false,
		AutoStopMachines:   true,
		AutoStartMachines:  true,
		MinMachinesRunning: 1,
		Concurrency: Concurrency{
			Type:      "requests",
			SoftLimit: 1_000_000_000,
			HardLimit: 1_000_000_000,
		},
		Ports: []Port{
			{
				Handlers:   []string{"http"},
				Port:       80,
				ForceHttps: true,
			},
			{
				Handlers: []string{"tls", "http"},
				Port:     443,
			},
		},
	}
}

func NewDefaultLaunchParams(
	appName string,
	orgSlug string,
) []string {
	args := []string{
		"--ha=false",
		"--auto-confirm",
		"--now",
		"--copy-config",
		"--name",
		appName,
	}

	if orgSlug != "" {
		args = append(args, "--org", orgSlug)
	}

	return args
}

func NewDefaultDeployParams() []string {
	args := []string{
		"--ha=false",
	}

	return args
}
