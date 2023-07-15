package model

import (
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/util/util_cfg_merge"
	"github.com/gigurra/flycd/internal/flycd/util/util_cvt"
	"gopkg.in/yaml.v3"
)

type CommonAppConfig struct {
	AppDefaults      map[string]any `yaml:"app_defaults" toml:"app_defaults"`   // default yaml tree for all apps
	AppSubstitutions map[string]any `yaml:"substitutions" toml:"substitutions"` // raw text substitution regexes
	AppOverrides     map[string]any `yaml:"app_overrides" toml:"app_overrides"` // yaml overrides for all apps
}

func (c CommonAppConfig) Plus(other CommonAppConfig) CommonAppConfig {
	return CommonAppConfig{
		AppDefaults:      util_cfg_merge.Merge(c.AppDefaults, other.AppDefaults),
		AppOverrides:     util_cfg_merge.Merge(c.AppOverrides, other.AppOverrides),
		AppSubstitutions: util_cfg_merge.Merge(c.AppSubstitutions, other.AppSubstitutions),
	}
}

func (c CommonAppConfig) MakeAppConfig(appYaml []byte) (AppConfig, map[string]any, error) {

	untyped := make(map[string]any)

	// TODO: Use common values

	err := yaml.Unmarshal(appYaml, &untyped)
	if err != nil {
		return AppConfig{}, untyped, fmt.Errorf("error unmarshalling app.yaml: %w", err)
	}

	typed, err := util_cvt.MapYamlToStruct[AppConfig](untyped)
	if err != nil {
		return typed, untyped, fmt.Errorf("error converting untyped app.yaml to typed: %w", err)
	}

	err = typed.Validate()
	if err != nil {
		return typed, untyped, fmt.Errorf("error validating app.yaml: %w", err)
	}

	return typed, untyped, nil
}
