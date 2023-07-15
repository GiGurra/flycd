package model

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestCommonAppConfig_MakeAppConfig(t *testing.T) {

	yamlConf := `
app: app1
org: test-org
primary_region: arn
extra_regions:
  - amx
deploy_params:
  - "--ok"
source:
  type: local
`

	commonCfg := CommonAppConfig{
		AppDefaults: map[string]any{
			"foo": "bar",
			"org": "fancy-org",
		},
		AppSubstitutions: map[string]any{
			"arn": "blarn",
		},
		AppOverrides: map[string]any{
			"foo":           "bar",
			"extra_regions": []string{"ams"},
			"deploy_params": []string{"--busted"},
		},
	}

	appCfgTyped, appCfgUntyped, err := commonCfg.MakeAppConfig([]byte(yamlConf), false)
	if err != nil {
		t.Fatalf("MakeAppConfig failed: %v", err)
	}

	wantedTyped := AppConfig{
		App:           "app1",
		Org:           "test-org",
		PrimaryRegion: "blarn",
		ExtraRegions:  []string{"ams"},
		DeployParams:  []string{"--busted"},
		Source: Source{
			Type: "local",
		},
	}

	wantedUntyped := map[string]any{
		"app":            "app1",
		"org":            "test-org",
		"primary_region": "blarn",
		"foo":            "bar",
		"extra_regions":  []string{"ams"},
		"deploy_params":  []string{"--busted"},
		"source": map[string]any{
			"type": "local",
		},
	}

	if diff := cmp.Diff(wantedTyped, appCfgTyped); diff != "" {
		t.Fatalf("MakeAppConfig failed, typed diff: %v", diff)
	}

	if diff := cmp.Diff(wantedUntyped, appCfgUntyped); diff != "" {
		t.Fatalf("MakeAppConfig failed, untyped diff: %v", diff)
	}
}
