package model

import (
	"fmt"
	"regexp"
)

type ProjectConfig struct {
	Project string       `yaml:"project" toml:"project"` // Name Required. Unique name of the project
	Source  Source       `yaml:"source" toml:"source"`   // Source Required. Where the app configs of the project are located
	Common  CommonParams `yaml:"common" toml:"common"`   // Common Optional. Common config for all apps in the project
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
