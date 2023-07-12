package model

import "time"

type DeployConfig struct {
	Force             bool
	Retries           int
	AttemptTimeout    time.Duration
	AbortOnFirstError bool
}

func NewDeployConfig() DeployConfig {
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
