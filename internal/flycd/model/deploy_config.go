package model

import (
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/util/util_cvt"
	"gopkg.in/yaml.v3"
	"time"
)

type CommonParams struct {
	AppDefaults      map[string]any    `yaml:"app_defaults" toml:"app_defaults"`   // default yaml tree for all apps
	AppSubstitutions map[string]string `yaml:"substitutions" toml:"substitutions"` // raw text substitution regexes
	AppOverrides     map[string]any    `yaml:"app_overrides" toml:"app_overrides"` // yaml overrides for all apps
}

func (c CommonParams) MakeAppConfig(appYaml []byte) (AppConfig, map[string]any, error) {

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

type DeployConfig struct {
	Force             bool
	Retries           int
	AttemptTimeout    time.Duration
	AbortOnFirstError bool
	CommonAppCfg      CommonParams
}

func NewDefaultDeployConfig() DeployConfig {
	return DeployConfig{
		Force:             false,
		Retries:           2,
		AttemptTimeout:    5 * time.Minute,
		AbortOnFirstError: true,
	}
}

func (c DeployConfig) WithAbortOnFirstError(state ...bool) DeployConfig {
	if len(state) > 0 {
		c.AbortOnFirstError = state[0]
	} else {
		c.AbortOnFirstError = true
	}
	return c
}

func (c DeployConfig) WithForce(force ...bool) DeployConfig {
	if len(force) > 0 {
		c.Force = force[0]
	} else {
		c.Force = true
	}
	return c
}

func (c DeployConfig) WithRetries(retries ...int) DeployConfig {
	if len(retries) > 0 {
		c.Retries = retries[0]
	} else {
		c.Retries = 5
	}
	return c
}

func (c DeployConfig) WithAttemptTimeout(timeout ...time.Duration) DeployConfig {
	if len(timeout) > 0 {
		c.AttemptTimeout = timeout[0]
	} else {
		c.AttemptTimeout = 5 * time.Minute
	}
	return c
}
