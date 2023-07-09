package flyctl

import (
	"flycd/internal/flycd/util/util_cmd"
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

type StoreSecretCmd struct {
	appName     string
	secretName  string
	secretValue string
	accessToken string
}

func StoreSecret(cmd StoreSecretCmd) error {

	// Initialize a shell command for creating an org token with flyctl
	// This operation is interactive, so we need to forward stdin, stdout, and stderr
	// so the user can interact with flyctl

	args := []string{
		"secrets",
		"set",
		fmt.Sprintf(`"%s"="%s"`, cmd.secretName, cmd.secretValue),
	}

	if cmd.appName != "" {
		args = append(args, "-a", cmd.appName)
	}

	if cmd.accessToken != "" {
		args = append(args, "-t", cmd.accessToken)
	}

	err := util_cmd.NewCommandA("flyctl", args...).RunStreamedPassThrough()
	if err != nil {
		return fmt.Errorf("error running flyctl secrets set for '%s': %w", cmd.secretName, err)
	}

	return nil
}
