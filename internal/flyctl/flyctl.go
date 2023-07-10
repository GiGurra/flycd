package flyctl

import (
	"context"
	"encoding/json"
	"flycd/internal/flycd/util/util_cmd"
	"fmt"
	"github.com/samber/lo"
	"os"
	"os/exec"
	"strings"
	"time"
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

	return strings.TrimSpace(lines[iLineToken]), nil
}

type StoreSecretCmd struct {
	AppName     string
	SecretName  string
	SecretValue string
}

type ExistsSecretCmd struct {
	AppName    string
	SecretName string
}

type flySecretListItem struct {
	Name      string    `json:"Name"`
	Digest    string    `json:"Digest"`
	CreatedAt time.Time `json:"CreatedAt"`
}

func ExistsSecret(ctx context.Context, cmd ExistsSecretCmd) (bool, error) {

	if cmd.SecretName == "" {
		return false, fmt.Errorf("secret name cannot be empty")
	}

	args := []string{
		"secrets",
		"list",
		"--json",
	}

	if cmd.AppName != "" {
		args = append(args, "-a", cmd.AppName)
	}

	res, err := util_cmd.
		NewCommandA("flyctl", args...).
		WithTimeout(10 * time.Second).
		WithTimeoutRetries(10).
		Run(ctx)
	if err != nil {
		return false, fmt.Errorf("error running flyctl secrets list for '%s': %w", cmd.AppName, err)
	}

	// Parse strResp as json array of flySecretListItem
	var secrets []flySecretListItem
	err = json.Unmarshal([]byte(res.StdOut), &secrets)
	if err != nil {
		return false, fmt.Errorf("error parsing flyctl secrets list for '%s': %w", cmd.AppName, err)
	}

	return lo.ContainsBy(secrets, func(item flySecretListItem) bool {
		return item.Name == cmd.SecretName
	}), nil
}

func StoreSecret(ctx context.Context, cmd StoreSecretCmd) error {

	if cmd.SecretName == "" {
		return fmt.Errorf("secret name cannot be empty")
	}

	if cmd.SecretValue == "" {
		return fmt.Errorf("secret value cannot be empty")
	}

	args := []string{
		"secrets",
		"set",
		fmt.Sprintf(`%s=%s`, cmd.SecretName, cmd.SecretValue),
	}

	if cmd.AppName != "" {
		args = append(args, "-a", cmd.AppName)
	}

	_, err := util_cmd.
		NewCommandA("flyctl", args...).
		WithTimeout(240 * time.Second).
		WithTimeoutRetries(5).
		WithStdLogging().
		Run(ctx)
	if err != nil {
		return fmt.Errorf("error running flyctl secrets set for '%s': %w", cmd.SecretName, err)
	}

	return nil
}
