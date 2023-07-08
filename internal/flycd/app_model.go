package flycd

import (
	"fmt"
	"regexp"
)

type AppConfig struct {
	App          string            `yaml:"app"`
	Source       Source            `yaml:"source"`
	Services     []Service         `yaml:"services"`
	LaunchParams []string          `yaml:"launch_params"`
	DeployParams []string          `yaml:"deploy_params"`
	Env          map[string]string `yaml:"env"`
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

	return nil
}

type Source struct {
	Repo string     `yaml:"repo"`
	Path string     `yaml:"path"`
	Ref  string     `yaml:"ref"`
	Type SourceType `yaml:"type"`
}

type SourceType string

const (
	SourceTypeGit    SourceType = "git"
	SourceTypeLocal  SourceType = "local"
	SourceTypeDocker SourceType = "docker"
)

func (s *Source) Validate() error {

	switch s.Type {
	case SourceTypeGit:
		if s.Repo == "" {
			return fmt.Errorf("repo is required")
		}
	case SourceTypeLocal:
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

type Ports struct {
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
	Ports              []Ports     `yaml:"ports"`
}
