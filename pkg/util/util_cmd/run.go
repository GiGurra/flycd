package util_cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/gigurra/flycd/pkg/util/util_context"
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
	Combined string
	Err      error
	Attempts int
}

func defaultTimeout() time.Duration {
	return 5 * time.Minute
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

func (c Command) WithExtraArgs(args ...string) Command {
	c.Args = append(c.Args, args...)
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

func (c Command) WithStdLogging() Command {
	c.LogStdErr = true
	c.LogStdOut = true
	return c
}

func (c Command) WithStdErrLogging() Command {
	c.LogStdErr = true
	return c
}

func (c Command) WithStdOutLogging() Command {
	c.LogStdOut = true
	return c
}

func (c Command) Run(ctx context.Context) (CommandResult, error) {

	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}
	combinedBuffer := &bytes.Buffer{}
	attempts := 0

	// This channel is used to signal that the timeout should be reset
	resetChan := make(chan any, 1)
	defer close(resetChan)

	err := c.withRetries(ctx, resetChan, func(cmd *exec.Cmd) error {

		// Reset these each time, because they could internally
		attempts++
		stdoutBuffer = &bytes.Buffer{}
		stderrBuffer = &bytes.Buffer{}
		combinedBuffer = &bytes.Buffer{}
		// create a writer that writes to buffer, but also sends a signal to reset the timeout
		combinedWriter := util_context.NewResetWriterCh(combinedBuffer, resetChan)

		cmd.Stdin = os.Stdin
		if c.LogStdOut {
			cmd.Stdout = io.MultiWriter(os.Stdout, stdoutBuffer, combinedWriter)
		} else {
			cmd.Stdout = io.MultiWriter(stdoutBuffer, combinedWriter)
		}
		if c.LogStdErr {
			cmd.Stderr = io.MultiWriter(os.Stderr, stderrBuffer, combinedWriter)
		} else {
			cmd.Stderr = io.MultiWriter(stderrBuffer, combinedWriter)
		}

		return cmd.Run()
	})

	stdout := stdoutBuffer.String()
	stderr := stderrBuffer.String()
	combined := combinedBuffer.String()

	if err != nil {
		err = fmt.Errorf("command failed: %w\nstderr:%s", err, stderr)
	}

	return CommandResult{
		StdOut:   stdout,
		StdErr:   stderr,
		Combined: combined,
		Err:      err,
		Attempts: attempts,
	}, err
}

func (c Command) withRetries(ctx context.Context, recvSignal <-chan any, processor func(cmd *exec.Cmd) error) error {

	c.logBeforeRun()

	for i := 0; i <= c.TimeoutRetries; i++ {

		ctx := ctx // needed so we don't cancel the parent context

		// Every retry needs its own timeout context
		err := func() error {
			if c.Timeout > 0 {
				var cancel context.CancelFunc
				var resetTimeout util_context.ResetFunc
				ctx, cancel, resetTimeout = util_context.WithTimeoutAndReset(ctx, c.Timeout)
				defer cancel()
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case _, ok := <-recvSignal:
							if !ok {
								return
							}
							resetTimeout()
						}
					}
				}()
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

			if strings.Contains(err.Error(), "context deadline exceeded") {
				fmt.Printf("timeout (context deadline exceeded) running util_cmd for %s, attempt %d/%d \n", c.App, i+1, c.TimeoutRetries+1)
				continue
			}

			return fmt.Errorf("error running util_cmd %s \n %s: %w", c.App, err.Error(), err)
		}

		return nil

	}
	return fmt.Errorf("error running util_cmd %s \n %s: %w", c.App, "timeout", context.DeadlineExceeded)
}
