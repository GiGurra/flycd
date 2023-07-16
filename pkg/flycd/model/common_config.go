package model

import (
	"fmt"
	"github.com/gigurra/flycd/pkg/flycd/util/util_cfg_merge"
	"github.com/gigurra/flycd/pkg/flycd/util/util_cvt"
	"gopkg.in/yaml.v3"
	"regexp"
)

type CommonAppConfig struct {
	AppDefaults      map[string]any `yaml:"app_defaults" toml:"app_defaults"`   // default yaml tree for all apps
	AppSubstitutions map[string]any `yaml:"substitutions" toml:"substitutions"` // raw text substitution regexes
	AppOverrides     map[string]any `yaml:"app_overrides" toml:"app_overrides"` // yaml overrides for all apps
}

func (c CommonAppConfig) Plus(other CommonAppConfig) CommonAppConfig {
	return CommonAppConfig{
		AppDefaults:      util_cfg_merge.MergeMaps(c.AppDefaults, other.AppDefaults, mergeKeys...),
		AppSubstitutions: util_cfg_merge.MergeMaps(c.AppSubstitutions, other.AppSubstitutions),
		AppOverrides:     util_cfg_merge.MergeMaps(c.AppOverrides, other.AppOverrides, mergeKeys...),
	}
}

var mergeKeys = []string{"internal_port", "id", "port", "name", "source"}

func (c CommonAppConfig) MakeAppConfig(appYaml []byte, validate ...bool) (AppConfig, map[string]any, error) {

	untypedLocal := make(map[string]any)

	// Copy the bytes to a new slice to avoid problems when doing regex matching
	appYaml = append([]byte{}, appYaml...)

	// Run all substitutions
	for from, to := range c.AppSubstitutions {
		regex, err := regexp.Compile(from)
		if err != nil {
			return AppConfig{}, untypedLocal, fmt.Errorf("error compiling common substitution regex '%s': %w", from, err)
		}
		stringTo := fmt.Sprintf("%v", to)
		appYaml = regex.ReplaceAll(appYaml, []byte(stringTo))
	}

	err := yaml.Unmarshal(appYaml, &untypedLocal)
	if err != nil {
		return AppConfig{}, untypedLocal, fmt.Errorf("error unmarshalling app.yaml: %w", err)
	}

	untyped := map[string]any{}
	untyped = util_cfg_merge.MergeMaps(untyped, c.AppDefaults)
	untyped = util_cfg_merge.MergeMaps(untyped, untypedLocal)
	untyped = util_cfg_merge.MergeMaps(untyped, c.AppOverrides)

	typed, err := util_cvt.MapYamlToStruct[AppConfig](untyped)
	if err != nil {
		return typed, untyped, fmt.Errorf("error converting untyped app.yaml to typed: %w", err)
	}

	if len(validate) == 0 || validate[0] {
		err = typed.Validate()
		if err != nil {
			return typed, untyped, fmt.Errorf("error validating app.yaml: %w", err)
		}
	}

	return typed, untyped, nil
}
