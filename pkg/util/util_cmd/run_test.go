package util_cmd

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type Check[T any] interface {
	Name() string
	Apply(t T) error
}

type check[T any] struct {
	name  string
	apply func(t T) error
}

func (c check[T]) Name() string {
	return c.name
}

func (c check[T]) Apply(t T) error {
	return c.apply(t)
}

func NewCheck[T any](name string, apply func(t T) error) Check[T] {
	return check[T]{
		name:  name,
		apply: apply,
	}
}

var errorIsNilCheck = NewCheck[CommandResult]("error is nil", func(result CommandResult) error {
	if result.Err != nil {
		return fmt.Errorf("expected no error, got %v", result.Err)
	}
	return nil
})

var stdOutNonEmptyCheck = NewCheck[CommandResult]("StdOut not empty", func(result CommandResult) error {
	if result.StdOut == "" {
		return errors.New("empty StdOut")
	}
	return nil
})

func TestCommand_Run(t *testing.T) {
	tests := []struct {
		name   string
		cmd    Command
		checks []Check[CommandResult]
	}{
		{
			name:   "simple ls command",
			cmd:    NewCommand("ls", "-la").WithTimeout(5 * time.Second),
			checks: []Check[CommandResult]{errorIsNilCheck, stdOutNonEmptyCheck},
		},
		{
			name: "timing out command",
			cmd:  NewCommand("sleep", "10").WithTimeout(1 * time.Second),
			checks: []Check[CommandResult]{
				NewCheck[CommandResult]("error is context.DeadlineExceeded", func(result CommandResult) error {
					if result.Err == nil {
						return errors.New("expected error")
					}
					if !errors.Is(result.Err, context.DeadlineExceeded) {
						return fmt.Errorf("expected error to be context.DeadlineExceeded, got %v", result.Err)
					}
					return nil
				}),
			},
		},
		{
			name:   "command writing output every second for 4 seconds does not time out",
			cmd:    NewCommand("bash", "-c", "for i in {1..4}; do echo $i; sleep 1; done").WithTimeout(2 * time.Second),
			checks: []Check[CommandResult]{errorIsNilCheck, stdOutNonEmptyCheck},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdResult, _ := tt.cmd.Run(context.Background())

			for _, check := range tt.checks {
				fmt.Printf(" - Running check '%v'...", check.Name())
				if err := check.Apply(cmdResult); err != nil {
					t.Errorf("Command.Run() check '%v' failed: %v", check.Name(), err)
				}
				fmt.Printf("OK \n")
			}
		})
	}
}
