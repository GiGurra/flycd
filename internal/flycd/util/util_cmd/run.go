package util_cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Command struct {
	Cwd  string
	App  string
	Args []string
	// You can have either a custom context or a timeout, not both
	Timeout        time.Duration
	TimeoutRetries int
	LogStdOut      bool
	LogStdErr      bool
	// debug functionality
	LogInput bool
}

type CommandResult struct {
	StdOut   string
	StdErr   string
	err      error
	Attempts int
}

func defaultTimeout() time.Duration {
	return 5 * time.Minute
}

func getAccessToken(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	token, ok := ctx.Value("FLY_ACCESS_TOKEN").(string)
	if !ok {
		return ""
	}
	return token
}

func NewCommand(appAndArgs ...string) Command {

	result := Command{
		Cwd:     ".",
		Timeout: defaultTimeout(),
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
		Cwd:     ".",
		App:     app,
		Args:    args,
		Timeout: defaultTimeout(),
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

func (c Command) WithTimeoutRetries(n int) Command {
	c.TimeoutRetries = n
	return c
}

func (c Command) WithLogging(args ...bool) Command {
	if len(args) > 0 {
		c.LogInput = args[0]
	} else {
		c.LogInput = true
	}
	return c
}

func (c Command) WithTimeout(timeout time.Duration) Command {
	c.Timeout = timeout
	return c
}

func (c Command) logBeforeRun() {
	if c.LogInput {
		if c.App == "sh" && len(c.Args) > 0 && c.Args[0] == "-c" {
			fmt.Printf("%s$ %s\n", c.Cwd, strings.Join(c.Args[1:], " "))
		} else {
			fmt.Printf("%s$ %s %s\n", c.Cwd, c.App, strings.Join(c.Args, " "))
		}
	}
}

func (c Command) RunStreamAndCapture(ctx context.Context) CommandResult {

	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}
	attempts := 0

	err := c.doRun(ctx, func(cmd *exec.Cmd) error {

		// Reset these each time, because they could internally
		attempts++
		stdoutBuffer = &bytes.Buffer{}
		stderrBuffer = &bytes.Buffer{}

		cmd.Stdin = os.Stdin
		if c.LogStdOut {
			cmd.Stdout = io.MultiWriter(os.Stdout, stdoutBuffer)
		} else {
			cmd.Stdout = stdoutBuffer
		}
		if c.LogStdErr {
			cmd.Stderr = io.MultiWriter(os.Stderr, stderrBuffer)
		} else {
			cmd.Stderr = stderrBuffer
		}

		return cmd.Run()
	})

	stdout := stdoutBuffer.String()
	stderr := stderrBuffer.String()

	return CommandResult{
		StdOut:   stdout,
		StdErr:   stderr,
		err:      err,
		Attempts: attempts,
	}
}

func (c Command) Run(ctx context.Context) (string, error) {

	res := c.RunStreamAndCapture(ctx)

	return res.StdOut, res.err
}

func (c Command) RunStreamedPassThrough(ctx context.Context) error {

	c.LogStdOut = true
	c.LogStdErr = true

	res := c.RunStreamAndCapture(ctx)

	return res.err
}

func (c Command) doRun(ctx context.Context, processor func(cmd *exec.Cmd) error) error {

	c.logBeforeRun()

	accessToken := getAccessToken(ctx)
	if accessToken != "" && (c.App == "flyctl" || c.App == "fly") {
		c.Args = append(c.Args, "--access-token", accessToken)
	}

	for i := 0; i <= c.TimeoutRetries; i++ {

		ctx := ctx // needed so we don't cancel the parent context

		err := func() error {
			if c.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, c.Timeout)
				defer cancel()
			}

			cmd := exec.CommandContext(ctx, c.App, c.Args...)
			cmd.Dir = c.Cwd

			return processor(cmd)
		}()
		if err != nil {

			if errors.Is(err, context.DeadlineExceeded) {
				fmt.Printf("timeout (go context.DeadlineExceeded) running util_cmd for %s, attempt %d/%d \n", c.App, i+1, c.TimeoutRetries+1)
				continue
			}

			if strings.Contains(err.Error(), "signal: killed") {
				fmt.Printf("timeout (signal: killed) running util_cmd for %s, attempt %d/%d \n", c.App, i+1, c.TimeoutRetries+1)
				continue
			}

			if strings.Contains(err.Error(), "request returned non-2xx status, 504") {
				fmt.Printf("timeout (http 504 from fly.io) running util_cmd for %s, attempt %d/%d \n", c.App, i+1, c.TimeoutRetries+1)
				continue
			}

			stdErr := ""
			if exitErr, ok := err.(*exec.ExitError); ok {
				stdErr = string(exitErr.Stderr)
			}
			return fmt.Errorf("error running util_cmd %s \n %s: %w", c.App, stdErr, err)
		}

		return nil

	}
	return fmt.Errorf("error running util_cmd %s \n %s: %w", c.App, "timeout", context.DeadlineExceeded)
}
