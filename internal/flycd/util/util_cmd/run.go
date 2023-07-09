package util_cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Command struct {
	Cwd  string
	App  string
	Args []string
	Ctx  context.Context
}

func NewCommand(appAndArgs ...string) Command {
	result := Command{
		Cwd: ".",
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

func (c Command) WithContext(ctx context.Context) Command {
	c.Ctx = ctx
	return c
}

func (c Command) Run() (string, error) {
	return Run(c.Ctx, c.Cwd, c.App, c.Args...)
}

func (c Command) RunStreamedPassThrough() error {
	return RunStreamedPassThrough(c.Ctx, c.Cwd, c.App, c.Args...)
}

func Run(ctx context.Context, cwd string, command string, args ...string) (string, error) {

	if command == "sh" && len(args) > 0 && args[0] == "-c" {
		fmt.Printf("%s$ %s\n", cwd, strings.Join(args[1:], " "))
	} else {
		fmt.Printf("%s$ %s %s\n", cwd, command, strings.Join(args, " "))
	}

	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {

		stdErr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stdErr = string(exitErr.Stderr)
		}
		return string(out), fmt.Errorf("error running util_cmd %s \n %s: %w", command, stdErr, err)
	}

	return string(out), nil
}

func RunStreamedPassThrough(ctx context.Context, cwd string, command string, args ...string) error {

	if command == "sh" && len(args) > 0 && args[0] == "-c" {
		fmt.Printf("%s$ %s\n", cwd, strings.Join(args[1:], " "))
	} else {
		fmt.Printf("%s$ %s %s\n", cwd, command, strings.Join(args, " "))
	}

	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = cwd

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {

		stdErr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stdErr = string(exitErr.Stderr)
		}
		return fmt.Errorf("error running util_cmd %s \n %s: %w", command, stdErr, err)
	}

	return nil
}
