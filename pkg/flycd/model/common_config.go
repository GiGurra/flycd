package model

import (
	"fmt"
	"github.com/gigurra/flycd/pkg/flycd/util/util_cfg_merge"
	"github.com/gigurra/flycd/pkg/flycd/util/util_cvt"
	"gopkg.in/yaml.v3"
	"regexp"
)

// CommonAppConfig is configuration defined in project.yaml files that applies to all apps in the project
type CommonAppConfig struct {
	AppDefaults      map[string]any `yaml:"app_defaults" toml:"app_defaults"`   // default yaml tree for all apps
	AppSubstitutions map[string]any `yaml:"substitutions" toml:"substitutions"` // raw text substitution regexes
	AppOverrides     map[string]any `yaml:"app_overrides" toml:"app_overrides"` // yaml overrides for all apps
}

// Plus merges two CommonAppConfig's, used when traversing the project tree with projects-in-projects.
func (c CommonAppConfig) Plus(other CommonAppConfig) CommonAppConfig {
	return CommonAppConfig{
		AppDefaults:      util_cfg_merge.MergeMaps(c.AppDefaults, other.AppDefaults),
		AppSubstitutions: util_cfg_merge.MergeMaps(c.AppSubstitutions, other.AppSubstitutions),
		AppOverrides:     util_cfg_merge.MergeMaps(c.AppOverrides, other.AppOverrides),
	}
}

// MakeAppConfig creates an AppConfig from a raw app.yaml file, applying all substitutions and overrides,
// from the parent projects and their CommonAppConfig's.
func (c CommonAppConfig) MakeAppConfig(appYaml []byte, validate ...bool) (AppConfig, map[string]any, error) {

	// Copy the bytes to a new slice to avoid modifying the original
	// when we start doing substitutions
	appYaml = append([]byte{}, appYaml...)

	// Run all substitutions
	for from, to := range c.AppSubstitutions {
		regex, err := regexp.Compile(from)
		if err != nil {
			return AppConfig{}, map[string]any{}, fmt.Errorf("error compiling common substitution regex '%s': %w", from, err)
		}
		stringTo := fmt.Sprintf("%v", to)
		appYaml = regex.ReplaceAll(appYaml, []byte(stringTo))
	}

	// Unmarshal the app.yaml into a map[string]any
	cfgInFile := make(map[string]any)
	err := yaml.Unmarshal(appYaml, &cfgInFile)
	if err != nil {
		return AppConfig{}, cfgInFile, fmt.Errorf("error unmarshalling app.yaml: %w", err)
	}

	// Combine all the configuration sources into one map
	untyped := map[string]any{}
	untyped = util_cfg_merge.MergeMaps(untyped, c.AppDefaults)
	untyped = util_cfg_merge.MergeMaps(untyped, cfgInFile)
	untyped = util_cfg_merge.MergeMaps(untyped, c.AppOverrides)

	// Convert the map to a typed AppConfig
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

// Maybe later we will use this
// var mergeKeys = []string{"internal_port", "id", "port", "name", "source"}
// var mergeConf = util_cfg_merge.MergeConf{...}
