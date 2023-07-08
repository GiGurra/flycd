package flyctl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func CreateOrgToken(orgSlug string) (string, error) {

	// Initialize a shell command for creating an org token with flyctl
	// This operation is interactive, so we need to forward stdin, stdout, and stderr
	// so the user can interact with flyctl

	// Run the command
	cmd := exec.Command("flyctl", "tokens", "create", "org", orgSlug)

	cmd.Stderr = os.Stderr

	// Run the command
	stdOut, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error running flyctl tokens create org: %w", err)
	}

	// split stdout by lines, in an os agnostic way
	lines := strings.Split(string(stdOut), "\n")

	// find the first line starting with "Fly"
	iLineToken := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "Fly") {
			iLineToken = i
			break
		}
	}

	// if we didn't find a line starting with "Fly", return an error
	if iLineToken == -1 {
		return "", fmt.Errorf("error parsing flyctl tokens create org output")
	}

	return lines[iLineToken], nil
}
