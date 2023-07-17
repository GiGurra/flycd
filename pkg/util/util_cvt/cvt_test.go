package util_cvt

import (
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
	"testing"
)

var someAppConf = `
app: &app some-application
#http_service:
#  auto_start_machines: true
#  auto_stop_machines: true
#  force_https: true
#  internal_port: 80
#  min_machines_running: 0
primary_region: &primary_region arn
source:
  type: local

org: &org personal

vm_size: &vm_size "shared-cpu-1x"

# Modify to your needs. By default, we will create a new fly.io
# app without any user interaction/confirmation.
# For the most simple apps, you probably don't need to modify these at all
launch_params:
  - "--ha=false"
  - "--auto-confirm"
  - "--now"
  - "--copy-config"
  - "--name"
  - *app
  - "--region"
  - *primary_region
  - "--org"
  - *org
  - "--vm-size"
  - *vm_size

# Modify to your needs. By default, we will deploy the fly.io
# app without any user interaction/confirmation.
# For the most simple apps, you probably don't need to modify these at all
deploy_params:
  - "--ha=false"
  - "--vm-size"
  - *vm_size

`

type FakeAppConfig struct {
	App           string         `yaml:"app" toml:"app"`
	HttpService   map[string]any `yaml:"http_service" toml:"http_service"`
	PrimaryRegion string         `yaml:"primary_region" toml:"primary_region"`
	Source        map[string]any `yaml:"source" toml:"source"`
	Org           string         `yaml:"org" toml:"org"`
	VmSize        string         `yaml:"vm_size" toml:"vm_size"`
	LaunchParams  []string       `yaml:"launch_params" toml:"launch_params"`
	DeployParams  []string       `yaml:"deploy_params" toml:"deploy_params"`
}

func TestMapYamlToStruct(t *testing.T) {
	var mapStringAny map[string]any
	err := yaml.Unmarshal([]byte(someAppConf), &mapStringAny)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	cfg, err := MapYamlToStruct[FakeAppConfig](mapStringAny)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.App != "some-application" {
		t.Fatalf("Unexpected app: %v", cfg.App)
	}

	if cfg.HttpService != nil {
		t.Fatalf("Unexpected http_service: %v", cfg.HttpService)
	}

	expectedConfig := FakeAppConfig{
		App:           "some-application",
		HttpService:   nil,
		PrimaryRegion: "arn",
		Source:        map[string]any{"type": "local"},
		Org:           "personal",
		VmSize:        "shared-cpu-1x",
		LaunchParams: []string{
			"--ha=false",
			"--auto-confirm",
			"--now",
			"--copy-config",
			"--name",
			"some-application",
			"--region",
			"arn",
			"--org",
			"personal",
			"--vm-size",
			"shared-cpu-1x",
		},
		DeployParams: []string{
			"--ha=false",
			"--vm-size",
			"shared-cpu-1x",
		},
	}

	diff := cmp.Diff(expectedConfig, cfg)
	if diff != "" {
		t.Fatalf("Unexpected config:\n%v", diff)
	}
}
