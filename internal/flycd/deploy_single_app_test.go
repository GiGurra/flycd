package flycd

import (
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/util/util_toml"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v3"
	"testing"
)

var srcYaml = `
############################
## Basic configuration
app: &app example-project1-local-app-foobar12341
org: &org personal

# Point where the code is.
source:
  type: "local"
  #path: somewhere/else/

############################
## Optional configuration
env:
  ENV: "development"

primary_region: &primary_region "arn" # default region for tests
services:
  - internal_port: 80
    protocol: "tcp"
    force_https: true
    auto_stop_machines: true
    auto_start_machines: true
    min_machines_running: 0
    concurrency:
      type: "requests"
      soft_limit: 200
      hard_limit: 250
    ports:
      - handlers: ["http"]
        port: 80
        force_https: true
      - handlers: ["tls", "http"]
        port: 443

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

# Modify to your needs. By default, we will deploy the fly.io
# app without any user interaction/confirmation.
# For the most simple apps, you probably don't need to modify these at all
deploy_params:
  - "--ha=false"
`

func TestConvertYaml2Toml(t *testing.T) {

	// Parse the yaml as a map
	mapData := make(map[string]any)
	err := yaml.Unmarshal([]byte(srcYaml), &mapData)
	if err != nil {
		t.Fatalf("Failed to parse yaml: %v", err)
	}

	fmt.Printf("Parsed yaml: %+v\n", mapData)
	backToYaml1, err := yaml.Marshal(mapData)
	if err != nil {
		t.Fatalf("Failed to convert yaml to yaml: %v", err)
	}

	// create an io.writer buffer to write the toml to

	// Convert the map to toml
	tomlStr, err := util_toml.Marshal(mapData)
	if err != nil {
		t.Fatalf("Failed to convert yaml to toml: %v", err)
	}

	fmt.Printf("Converted toml: \n%s\n", tomlStr)

	// Convert the toml back to yaml
	var mapData2 map[string]any
	err = util_toml.Unmarshal(tomlStr, &mapData2)
	if err != nil {
		t.Fatalf("Failed to convert toml to yaml: %v", err)
	}

	// Convert the map to yaml
	backToYaml2, err := yaml.Marshal(mapData2)
	if err != nil {
		t.Fatalf("Failed to convert toml to yaml: %v", err)
	}

	// Check that the maps are equal
	diff := cmp.Diff(backToYaml1, backToYaml2)
	if diff != "" {
		t.Fatalf("Maps are not equal: %v", diff)
	}
}
