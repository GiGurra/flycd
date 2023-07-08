package flycd

import (
	"fmt"
	"regexp"
)

type AppConfig struct {
	App           string            `yaml:"app"`
	Org           string            `yaml:"org"`
	PrimaryRegion string            `yaml:"primary_region"`
	Source        Source            `yaml:"source"`
	Services      []Service         `yaml:"services"`
	LaunchParams  []string          `yaml:"launch_params"`
	DeployParams  []string          `yaml:"deploy_params"`
	Env           map[string]string `yaml:"env"`
}

func (a *AppConfig) Validate() error {
	if a.App == "" {
		return fmt.Errorf("app name is required")
	}

	// only permit apps that are valid dns names
	const subdomainPrefixRegExp = `^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`

	subdomainPrefixChecker := regexp.MustCompile(subdomainPrefixRegExp)
	if !subdomainPrefixChecker.MatchString(a.App) {
		return fmt.Errorf("app name '%s' is not a valid subdomain prefix", a.App)
	}

	err := a.Source.Validate()
	if err != nil {
		return err
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

type Source struct {
	Repo   string     `yaml:"repo"`
	Path   string     `yaml:"path"`
	Ref    string     `yaml:"ref"`
	Type   SourceType `yaml:"type"`
	Inline string     `yaml:"inline"`
}

func NewInlineDockerFileSource(inline string) Source {
	return Source{
		Type:   SourceTypeInlineDockerFile,
		Inline: inline,
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

	if s.Type == SourceTypeGit && s.Ref == "" {
		return fmt.Errorf("ref is required for git source type")
	}

	return nil
}

type Concurrency struct {
	Type      string `yaml:"type"`
	SoftLimit int    `yaml:"soft_limit"`
	HardLimit int    `yaml:"hard_limit"`
}

type Port struct {
	Handlers   []string `yaml:"handlers"`
	Port       int      `yaml:"port"`
	ForceHttps bool     `yaml:"force_https"`
}

type Service struct {
	InternalPort       int         `yaml:"internal_port"`
	Protocol           string      `yaml:"protocol"`
	ForceHttps         bool        `yaml:"force_https"`
	AutoStopMachines   bool        `yaml:"auto_stop_machines"`
	AutoStartMachines  bool        `yaml:"auto_start_machines"`
	MinMachinesRunning int         `yaml:"min_machines_running"`
	Concurrency        Concurrency `yaml:"concurrency"`
	Ports              []Port      `yaml:"ports"`
}
