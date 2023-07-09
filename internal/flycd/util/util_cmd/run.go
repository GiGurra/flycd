package util_cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Command struct {
	Cwd     string
	App     string
	Args    []string
	Ctx     context.Context
	Logging bool
}

func NewCommand(appAndArgs ...string) Command {

	result := Command{
		Cwd: ".",
		Ctx: context.Background(),
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
		Cwd:  ".",
		App:  app,
		Args: args,
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
