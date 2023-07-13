package model

import (
	"fmt"
	"regexp"
)

type ProjectConfig struct {
	// Name Required. Name of the project
	Project string `yaml:"project" toml:"project"`
	// Source Required. Where the app configs of the project are located
	Source Source `yaml:"source" toml:"source"`
	// TODO: Implement the below
	// Org Optional. Default org to be used by all apps in the project
	//Org string `yaml:"org" toml:"org"`
	// PrimaryRegion Optional. Default region to be used by all apps in the project
	//PrimaryRegion string `yaml:"primary_region" toml:"primary_region"`
	// Env Optional. Default env vars to be used by all apps in the project
	//Env map[string]string `yaml:"env" toml:"env"`
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
