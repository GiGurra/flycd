package model

type ProjectConfig struct {
	// Name Required. Name of the project
	Name string `yaml:"name" toml:"name"`
	// Source Required. Where the app configs of the project are located
	Source Source `yaml:"source" toml:"source"`
	// Org Optional. Default org to be used by all apps in the project
	Org string `yaml:"org" toml:"org"`
	// PrimaryRegion Optional. Default region to be used by all apps in the project
	PrimaryRegion string `yaml:"primary_region" toml:"primary_region"`
	// Env Optional. Default env vars to be used by all apps in the project
	Env map[string]string `yaml:"env" toml:"env"`
}
