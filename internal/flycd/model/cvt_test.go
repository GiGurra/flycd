package model

import (
	"github.com/gigurra/flycd/internal/flycd/util/util_cvt"
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

func TestMapYamlToStruct(t *testing.T) {
	var mapStringAny map[string]any
	err := yaml.Unmarshal([]byte(someAppConf), &mapStringAny)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	cfg, err := util_cvt.MapYamlToStruct[AppConfig](mapStringAny)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if cfg.App != "some-application" {
		t.Fatalf("Unexpected app: %v", cfg.App)
	}

	if cfg.HttpService != nil {
		t.Fatalf("Unexpected http_service: %v", cfg.HttpService)
	}
}
