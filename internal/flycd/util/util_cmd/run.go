package util_cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func RunLocal(command string, args ...string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("unable to get current working directory: %w", err)
	}

	return Run(wd, command, args...)
}

func Run(cwd string, command string, args ...string) (string, error) {

	if command == "sh" && len(args) > 0 && args[0] == "-c" {
		fmt.Printf("%s$ %s\n", cwd, strings.Join(args[1:], " "))
	} else {
		fmt.Printf("%s$ %s %s\n", cwd, command, strings.Join(args, " "))
	}
	cmd := exec.Command(command, args...)
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

func RunStreamedPassThrough(cwd string, command string, args ...string) error {

	if command == "sh" && len(args) > 0 && args[0] == "-c" {
		fmt.Printf("%s$ %s\n", cwd, strings.Join(args[1:], " "))
	} else {
		fmt.Printf("%s$ %s %s\n", cwd, command, strings.Join(args, " "))
	}
	cmd := exec.Command(command, args...)
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
