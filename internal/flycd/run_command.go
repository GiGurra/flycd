package flycd

import (
	"fmt"
	"os/exec"
)

func runCommand(cwd string, command string, args ...string) (string, error) {
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
