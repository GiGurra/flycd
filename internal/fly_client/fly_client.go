package fly_client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/flycd/util/util_cmd"
	"github.com/gigurra/flycd/internal/flycd/util/util_work_dir"
	"github.com/samber/lo"
	"strconv"
	"strings"
	"time"
)

type FlyClient interface {
	CreateOrgToken(
		ctx context.Context,
		orgSlug string,
	) (string, error)

	ExistsSecret(
		ctx context.Context,
		cmd ExistsSecretCmd,
	) (bool, error)

	StoreSecret(
		ctx context.Context,
		cmd StoreSecretCmd,
	) error

	ExistsApp(
		ctx context.Context,
		name string,
	) (bool, error)

	GetDeployedAppConfig(
		ctx context.Context,
		name string,
	) (model.AppConfig, error)

	GetAppVolumes(
		ctx context.Context,
		name string,
	) ([]model.VolumeState, error)

	CreateNewApp(
		ctx context.Context,
		cfg model.AppConfig,
		tempDir util_work_dir.WorkDir,
		twoStep bool,
	) error

	DeployExistingApp(
		ctx context.Context,
		cfg model.AppConfig,
		tempDir util_work_dir.WorkDir,
		deployCfg model.DeployConfig,
	) error

	CreateVolume(
		ctx context.Context,
		app string,
		cfg model.VolumeConfig,
		region string,
	) (model.VolumeState, error)

	GetAppScale(
		ctx context.Context,
		app string,
	) ([]model.ScaleState, error)

	ExtendVolume(
		ctx context.Context,
		appName string,
		volumeId string,
		gb int,
	) error
}

type FlyClientImpl struct{}

func NewFlyClient() FlyClient {
	return &FlyClientImpl{}
}

var _ FlyClient = FlyClientImpl{}

func (c FlyClientImpl) CreateOrgToken(ctx context.Context, orgSlug string) (string, error) {

	result, err := util_cmd.
		NewCommandA("fly", "tokens", "create", "org", orgSlug).
		WithTimeout(10 * time.Second).
		WithTimeoutRetries(1).
		Run(ctx)

	if err != nil {
		return "", fmt.Errorf("error running 'fly tokens create org': %w", err)
	}

	// Run the command
	stdOut := result.StdOut

	// split stdout by lines, in an os agnostic way
	lines := strings.Split(stdOut, "\n")

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
		return "", fmt.Errorf("error parsing fly tokens create org output")
	}

	return strings.TrimSpace(lines[iLineToken]), nil
}

func (c FlyClientImpl) GetAppScale(
	ctx context.Context,
	app string,
) ([]model.ScaleState, error) {

	result, err := util_cmd.
		NewCommandA("fly", "scale", "show", "-a", app, "--json").
		WithTimeout(20 * time.Second).
		WithTimeoutRetries(2).
		Run(ctx)

	if err != nil {
		return nil, fmt.Errorf("error running 'fly scale show -a %s --json': %w", app, err)
	}

	var scaleStates []model.ScaleState
	err = json.Unmarshal([]byte(result.StdOut), &scaleStates)
	if err != nil {
		return nil, fmt.Errorf("error parsing fly scale show output for app %s: %w", app, err)
	}

	return scaleStates, nil
}

func (c FlyClientImpl) ExtendVolume(
	ctx context.Context,
	appName string,
	volumeId string,
	gb int,
) error {

	_, err := util_cmd.
		NewCommandA("fly", "volume", "extend", volumeId, "-a", appName, "-s", strconv.FormatInt(int64(gb), 10)).
		WithTimeout(60 * time.Second).
		WithTimeoutRetries(2).
		Run(ctx)

	if err != nil {
		return fmt.Errorf("error running 'fly volume extend %s -a %s': %w", volumeId, appName, err)
	}

	return nil
}

func (c FlyClientImpl) CreateVolume(
	ctx context.Context,
	app string,
	cfg model.VolumeConfig,
	region string,
) (model.VolumeState, error) {

	result, err := util_cmd.
		NewCommandA("fly", "volumes", "create", cfg.Name, "--region", region, "--app", app, "-y", "--json").
		WithTimeout(60 * time.Second).
		WithTimeoutRetries(0).
		Run(ctx)

	if err != nil {
		return model.VolumeState{}, fmt.Errorf("error running 'fly volumes create' for app %s: %w", app, err)
	}

	var volumeState model.VolumeState
	err = json.Unmarshal([]byte(result.StdOut), &volumeState)
	if err != nil {
		return model.VolumeState{}, fmt.Errorf("error parsing fly volumes create output for app %s: %w", app, err)
	}

	return volumeState, nil
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

func (c FlyClientImpl) ExistsSecret(ctx context.Context, cmd ExistsSecretCmd) (bool, error) {

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
		NewCommandA("fly", args...).
		WithTimeout(10 * time.Second).
		WithTimeoutRetries(5).
		Run(ctx)
	if err != nil {
		return false, fmt.Errorf("error running fly secrets list for '%s': %w", cmd.AppName, err)
	}

	// Parse strResp as json array of flySecretListItem
	var secrets []flySecretListItem
	err = json.Unmarshal([]byte(res.StdOut), &secrets)
	if err != nil {
		return false, fmt.Errorf("error parsing fly secrets list for '%s': %w", cmd.AppName, err)
	}

	return lo.ContainsBy(secrets, func(item flySecretListItem) bool {
		return item.Name == cmd.SecretName
	}), nil
}

func (c FlyClientImpl) StoreSecret(ctx context.Context, cmd StoreSecretCmd) error {

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
		NewCommandA("fly", args...).
		WithTimeout(240 * time.Second).
		WithTimeoutRetries(5).
		WithStdLogging().
		Run(ctx)
	if err != nil {
		return fmt.Errorf("error running fly secrets set for '%s': %w", cmd.SecretName, err)
	}

	return nil
}

func (c FlyClientImpl) ExistsApp(ctx context.Context, name string) (bool, error) {
	res, err := util_cmd.NewCommand("fly", "status", "-a", name).
		WithTimeout(10 * time.Second).
		WithTimeoutRetries(5).
		Run(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(res.Combined), "could not find app") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c FlyClientImpl) GetAppVolumes(
	ctx context.Context,
	name string,
) ([]model.VolumeState, error) {

	res, err := util_cmd.NewCommand("fly", "volumes", "list", "-a", name, "--json").
		WithTimeout(20 * time.Second).
		WithTimeoutRetries(5).
		Run(ctx)

	if err != nil {
		return []model.VolumeState{}, fmt.Errorf("error running fly volumes list for app %s: %w", name, err)
	}

	var volumes []model.VolumeState
	err = json.Unmarshal([]byte(res.StdOut), &volumes)
	if err != nil {
		return []model.VolumeState{}, fmt.Errorf("error parsing fly volumes list for app %s: %w", name, err)
	}

	return volumes, nil
}

func (c FlyClientImpl) GetDeployedAppConfig(ctx context.Context, name string) (model.AppConfig, error) {

	res, err := util_cmd.NewCommand("fly", "config", "show", "-a", name).
		WithTimeout(20 * time.Second).
		WithTimeoutRetries(5).
		Run(ctx)
	if err != nil {

		if strings.Contains(strings.ToLower(err.Error()), "no machines configured for this app") {
			return model.AppConfig{Env: map[string]string{}}, nil
		}

		return model.AppConfig{}, fmt.Errorf("error running fly config show for app %s: %w", name, err)
	}

	var deployedCfg model.AppConfig
	err = json.Unmarshal([]byte(res.StdOut), &deployedCfg)
	if err != nil {
		return model.AppConfig{}, fmt.Errorf("error unmarshalling fly config for app %s: %w", name, err)
	}

	return deployedCfg, nil
}

func (c FlyClientImpl) CreateNewApp(
	ctx context.Context,
	cfg model.AppConfig,
	tempDir util_work_dir.WorkDir,
	twoStep bool,
) error {
	allParams := append([]string{"launch"}, cfg.LaunchParams...)
	allParams = append(allParams, "--remote-only")
	if twoStep {
		allParams = append(allParams, "--build-only")
	}
	_, err := tempDir.
		NewCommand("fly", allParams...).
		WithTimeout(20 * time.Second).
		WithTimeoutRetries(5).
		WithStdLogging().
		Run(ctx)
	if err != nil {
		return fmt.Errorf("error creating app %s: %w", cfg.App, err)
	}
	return nil
}

func (c FlyClientImpl) DeployExistingApp(
	ctx context.Context,
	cfg model.AppConfig,
	tempDir util_work_dir.WorkDir,
	deployCfg model.DeployConfig,
) error {
	allParams := append([]string{"deploy"}, cfg.DeployParams...)
	allParams = append(allParams, "--remote-only", "--detach")

	_, err := tempDir.
		NewCommand("fly", allParams...).
		WithTimeout(deployCfg.AttemptTimeout).
		WithTimeoutRetries(deployCfg.Retries).
		WithStdLogging().
		Run(ctx)
	if err != nil {
		return fmt.Errorf("error deploying app %s: %w", cfg.App, err)
	}
	return nil
}
