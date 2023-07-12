package flycd

import (
	"fmt"
	"regexp"
)

type AppConfig struct {
	App           string            `yaml:"app" toml:"app"`
	Org           string            `yaml:"org" toml:"org"`
	PrimaryRegion string            `yaml:"primary_region" toml:"primary_region"`
	Source        Source            `yaml:"source" toml:"source"`
	Services      []Service         `yaml:"services" toml:"services"`
	LaunchParams  []string          `yaml:"launch_params" toml:"launch_params"`
	DeployParams  []string          `yaml:"deploy_params" toml:"deploy_params"`
	Env           map[string]string `yaml:"env" toml:"env"`
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

type GitRef struct {
	Branch string `yaml:"branch" toml:"branch"`
	Tag    string `yaml:"tag" toml:"tag"`
	Commit string `yaml:"commit" toml:"commit"`
}

func (g *GitRef) IsEmpty() bool {
	return g.Branch == "" && g.Tag == "" && g.Commit == ""
}

type Source struct {
	Repo   string     `yaml:"repo" toml:"repo"`
	Path   string     `yaml:"path" toml:"path"`
	Ref    GitRef     `yaml:"ref" toml:"ref"`
	Type   SourceType `yaml:"type" toml:"type"`
	Inline string     `yaml:"inline" toml:"inline"`
}

func NewInlineDockerFileSource(inline string) Source {
	return Source{
		Type:   SourceTypeInlineDockerFile,
		Inline: inline,
	}
}

func NewLocalFolderSource(path string) Source {
	return Source{
		Type: SourceTypeLocal,
		Path: path,
	}
}

func NewGitSource(url string) Source {
	return Source{
		Type: SourceTypeGit,
		Repo: url,
	}
}

type SourceType string

const (
	SourceTypeGit              SourceType = "git"
	SourceTypeLocal            SourceType = "local"
	SourceTypeDocker           SourceType = "docker"
	SourceTypeInlineDockerFile SourceType = "inline-docker-file"
)

func (s *Source) Validate() error {

	switch s.Type {
	case SourceTypeGit:
		if s.Repo == "" {
			return fmt.Errorf("repo is required")
		}
	case SourceTypeLocal:
	case SourceTypeInlineDockerFile:
		if s.Inline == "" {
			return fmt.Errorf("inline docker file is required")
		}
	case SourceTypeDocker:
		return fmt.Errorf("docker source type not implemented")
	default:
		return fmt.Errorf("invalid source type: %s", s.Type)
	}

	return nil
}

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
	Concurrency        Concurrency `yaml:"concurrency" toml:"concurrency"`
	Ports              []Port      `yaml:"ports" toml:"ports"`
}
