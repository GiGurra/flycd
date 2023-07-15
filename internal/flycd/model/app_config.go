package model

import (
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/util/util_math"
	"github.com/samber/lo"
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
}

func (a *AppConfig) Regions() []string {
	result := []string{}
	result = append(result, a.PrimaryRegion)
	result = append(result, a.ExtraRegions...)
	return lo.Uniq(result)
}

func (a *AppConfig) MinMachinesRunning() int {
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
			1,
		),
	)
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

	if !lo.Contains(a.ExtraRegions, a.PrimaryRegion) {
		a.ExtraRegions = append(a.ExtraRegions, a.PrimaryRegion)
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
		MinMachinesRunning: 0,
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
