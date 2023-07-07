package flycd

import (
	"fmt"
	"os/exec"
	"strings"
)

func runCommand(cwd string, command string, args ...string) (string, error) {

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
		return string(out), fmt.Errorf("error running command %s \n %s: %w", command, stdErr, err)
	}

	return string(out), nil
}
