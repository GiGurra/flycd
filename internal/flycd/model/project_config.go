package model

import (
	"fmt"
	"regexp"
)

type CommonValues struct {
	Global  map[string]any `yaml:"global" toml:"global"`
	Project map[string]any `yaml:"project" toml:"project"`
	App     map[string]any `yaml:"app" toml:"app"`
}

type ProjectConfig struct {
	// Name Required. Name of the project
	Project string `yaml:"project" toml:"project"`
	// Source Required. Where the app configs of the project are located
	Source   Source       `yaml:"source" toml:"source"`
	Defaults CommonValues `yaml:"defaults" toml:"defaults"`
	// Regex substitutions to be applied to all app configs, and child projects (everything recursively) in this project
	// These override the values in the child apps and child projects, if the regex matches
	Substitutions map[string]string `yaml:"substitutions" toml:"substitutions"`
	Overrides     CommonValues      `yaml:"overrides" toml:"overrides"`
}

func (cfg *ProjectConfig) Validate() error {
	if cfg.Project == "" {
		return fmt.Errorf("project name is required")
	}

	// only permit apps that are valid dns names
	const subdomainPrefixRegExp = `^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`

	subdomainPrefixChecker := regexp.MustCompile(subdomainPrefixRegExp)
	if !subdomainPrefixChecker.MatchString(cfg.Project) {
		return fmt.Errorf("project name '%s' is not a valid subdomain prefix", cfg.Project)
	}

	err := cfg.Source.Validate()
	if err != nil {
		return fmt.Errorf("project source is invalid: %w", err)
	}

	switch cfg.Source.Type {
	case SourceTypeLocal:
		//ok
	case SourceTypeGit:
		//ok
	default:
		return fmt.Errorf("project source type '%s' is invalid/not allowed", cfg.Source.Type)
	}

	return nil
}
