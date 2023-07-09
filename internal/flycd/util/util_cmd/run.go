package util_cmd

import (
	"context"
	"flycd/internal/globals"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Command struct {
	Cwd         string
	App         string
	Args        []string
	AccessToken string
	// You can have either a custom context or a timeout, not both
	Ctx     context.Context
	Timeout time.Duration
	// debug functionality
	Logging bool
}

func defaultTimeout() time.Duration {
	return 5 * time.Minute
}

func NewCommand(appAndArgs ...string) Command {

	result := Command{
		Cwd:         ".",
		Ctx:         context.Background(),
		Timeout:     defaultTimeout(),
		AccessToken: globals.GetAccessToken(),
	}

	if len(appAndArgs) > 0 {
		result.App = appAndArgs[0]
	}

	if len(appAndArgs) > 1 {
		result.Args = appAndArgs[1:]
	}

	return result
}
func NewCommandA(app string, args ...string) Command {
	result := Command{
		Cwd:         ".",
		App:         app,
		Args:        args,
		Ctx:         context.Background(),
		Timeout:     defaultTimeout(),
		AccessToken: globals.GetAccessToken(),
	}
	return result
}

func (c Command) WithCwd(cwd string) Command {
	c.Cwd = cwd
	return c
}

func (c Command) WithApp(app string, args ...string) Command {
	c.App = app
	c.Args = args
	return c
}

func (c Command) WithLogging(args ...bool) Command {
	if len(args) > 0 {
		c.Logging = args[0]
	} else {
		c.Logging = true
	}
	return c
}

func (c Command) WithContext(ctx context.Context) Command {
	c.Ctx = ctx
	return c
}

func (c Command) WithTimeout(timeout time.Duration) Command {
	c.Timeout = timeout
	return c
}

func (c Command) logBeforeRun() {
	if c.Logging {
		if c.App == "sh" && len(c.Args) > 0 && c.Args[0] == "-c" {
			fmt.Printf("%s$ %s\n", c.Cwd, strings.Join(c.Args[1:], " "))
		} else {
			fmt.Printf("%s$ %s %s\n", c.Cwd, c.App, strings.Join(c.Args, " "))
		}
	}
}

func (c Command) Run() (string, error) {

	c.logBeforeRun()

	if c.Timeout > 0 {
		var cancel context.CancelFunc
		c.Ctx, cancel = context.WithTimeout(c.Ctx, c.Timeout)
		defer cancel()
	}

	if c.AccessToken != "" && (c.App == "flyctl" || c.App == "fly") {
		c.Args = append(c.Args, "--access-token", c.AccessToken)
	}

	cmd := exec.CommandContext(c.Ctx, c.App, c.Args...)
	cmd.Dir = c.Cwd
	out, err := cmd.Output()
	if err != nil {

		stdErr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stdErr = string(exitErr.Stderr)
		}
		return string(out), fmt.Errorf("error running util_cmd %s \n %s: %w", c.App, stdErr, err)
	}

	return string(out), nil
}

func (c Command) RunStreamedPassThrough() error {

	c.logBeforeRun()

	if c.Timeout > 0 {
		var cancel context.CancelFunc
		c.Ctx, cancel = context.WithTimeout(c.Ctx, c.Timeout)
		defer cancel()
	}

	if c.AccessToken != "" && (c.App == "flyctl" || c.App == "fly") {
		c.Args = append(c.Args, "--access-token", c.AccessToken)
	}

	cmd := exec.CommandContext(c.Ctx, c.App, c.Args...)
	cmd.Dir = c.Cwd

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {

		stdErr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stdErr = string(exitErr.Stderr)
		}
		return fmt.Errorf("error running util_cmd %s \n %s: %w", c.App, stdErr, err)
	}

	return nil
}
