package fly_client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gigurra/flycd/pkg/domain/model"
	"github.com/gigurra/flycd/pkg/util/util_cmd"
	"github.com/gigurra/flycd/pkg/util/util_tab_table"
	"github.com/gigurra/flycd/pkg/util/util_work_dir"
	"github.com/samber/lo"
	"strconv"
	"strings"
	"time"
)

type AppListItem struct {
	Name string `json:"name"`
	Org  string `json:"org"`
}

type IpListItem struct {
	Id        string    `json:"ID"`
	Address   string    `json:"Address"`
	Type      string    `json:"Type"`
	Region    string    `json:"Region"`
	Network   string    `json:"Network"`
	CreatedAt time.Time `json:"CreatedAt"`
}

func (i IpListItem) IsPrivate() bool {
	return strings.Contains(strings.ToLower(i.Type), "private")
}

func (i IpListItem) Ipv() model.Ipv {
	if strings.Contains(strings.ToLower(i.Type), "v6") {
		return model.IpV6
	} else if strings.Contains(strings.ToLower(i.Type), "v4") {
		return model.IpV4
	} else {
		return model.IpVUkn
	}
}

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
		region string,
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

	ScaleApp(
		ctx context.Context,
		app string,
		region string,
		count int,
	) error

	SaveSecrets(
		ctx context.Context,
		app string,
		secrets []Secret,
		stage bool,
	) error

	ListApps(
		ctx context.Context,
	) ([]AppListItem, error)

	ListIps(
		ctx context.Context,
		app string,
	) ([]IpListItem, error)

	DeleteIp(
		ctx context.Context,
		app string,
		id string,
		address string,
	) error

	CreateIp(
		ctx context.Context,
		app string,
		ip model.IpConfig,
	) error
}

type FlyClientImpl struct{}

func NewFlyClient() FlyClient {
	return &FlyClientImpl{}
}

var _ FlyClient = FlyClientImpl{}

func (c FlyClientImpl) CreateIp(
	ctx context.Context,
	app string,
	ip model.IpConfig,
) error {

	allocateString := "allocate-v6"
	if ip.V == model.IpV4 {
		allocateString = "allocate-v4"
	}

	params := []string{"fly", "ips", allocateString, "-a", app}

	if ip.Private {
		params = append(params, "--private")
	}

	if ip.Region != "" {
		params = append(params, "--region", ip.Region)
	}

	if ip.Network != "" {
		params = append(params, "--network", ip.Network)
	}

	if ip.Org != "" {
		params = append(params, "--org", ip.Org)
	}

	if ip.Shared {
		params = append(params, "--shared")
	}

	_, err := util_cmd.
		NewCommand(params...).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithTimeout(1 * time.Minute).
		WithTimeoutRetries(1).
		Run(ctx)

	if err != nil {
		return fmt.Errorf("error allocating ip %+v for app %s: %w", ip, app, err)
	} else {
		return nil
	}
}

func (c FlyClientImpl) DeleteIp(
	ctx context.Context,
	app string,
	id string,
	address string,
) error {

	_, err := util_cmd.
		NewCommand("fly", "ips", "release", address, "-a", app).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithTimeout(1 * time.Minute).
		WithTimeoutRetries(1).
		Run(ctx)

	if err != nil {
		return fmt.Errorf("error releasing ip %s for app %s: %w", address, app, err)
	} else {
		return nil
	}
}

func (c FlyClientImpl) ListIps(ctx context.Context, app string) ([]IpListItem, error) {

	res, err := util_cmd.
		NewCommand("fly", "ips", "list", "-a", app, "--json").
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithTimeout(1 * time.Minute).
		WithTimeoutRetries(1).
		Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting ips list. Do you have a token loaded?: %w", err)
	}

	// Prob chang this to use json instead
	items := make([]IpListItem, 0)
	err = json.Unmarshal([]byte(res.StdOut), &items)
	if err != nil {
		return nil, fmt.Errorf("error parsing ips list: %w", err)
	}

	return items, nil
}

func (c FlyClientImpl) ListApps(ctx context.Context) ([]AppListItem, error) {

	// ensure we have a token loaded for the org we are monitoring
	res, err := util_cmd.
		NewCommand("fly", "apps", "list").
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithTimeout(2 * time.Minute).
		WithTimeoutRetries(1).
		Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting apps list. Do you have a token loaded?: %w", err)
	}

	// Prob chang this to use json instead
	appsTable, err := util_tab_table.ParseTable(res.StdOut)
	if err != nil {
		return nil, fmt.Errorf("error parsing apps list: %w", err)
	}

	result := make([]AppListItem, 0)
	for _, appRow := range appsTable.RowMaps {
		name := appRow["NAME"]
		org := appRow["OWNER"]

		result = append(result, AppListItem{
			Name: name,
			Org:  org,
		})
	}

	return result, nil
}

func (c FlyClientImpl) CreateOrgToken(ctx context.Context, orgSlug string) (string, error) {

	result, err := util_cmd.
		NewCommandA("fly", "tokens", "create", "org", orgSlug).
		WithExtraArgs(accessTokenArgs(ctx)...).
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
		WithExtraArgs(accessTokenArgs(ctx)...).
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

func (c FlyClientImpl) ScaleApp(
	ctx context.Context,
	app string,
	region string,
	count int,
) error {

	_, err := util_cmd.
		NewCommandA("fly", "scale", "count", strconv.FormatInt(int64(count), 10), "--app", app, "--region", region, "-y").
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithTimeout(120 * time.Second).
		WithTimeoutRetries(1).
		Run(ctx)

	if err != nil {
		return fmt.Errorf("error running 'fly scale count %d --app %s --region %s -y': %w", count, app, region, err)
	}

	return nil
}

func (c FlyClientImpl) ExtendVolume(
	ctx context.Context,
	appName string,
	volumeId string,
	gb int,
) error {

	_, err := util_cmd.
		NewCommandA("fly", "volume", "extend", volumeId, "-a", appName, "-s", strconv.FormatInt(int64(gb), 10)).
		WithExtraArgs(accessTokenArgs(ctx)...).
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
		NewCommandA("fly", "volumes", "create", cfg.Name, "--region", region, "-s", strconv.FormatInt(int64(cfg.SizeGb), 10), "--app", app, "-y", "--json").
		WithExtraArgs(accessTokenArgs(ctx)...).
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

type Secret struct {
	Name  string
	Value string
}

func (c FlyClientImpl) SaveSecrets(
	ctx context.Context,
	app string,
	secrets []Secret,
	stage bool,
) error {

	args := []string{"secrets", "set", "-a", app}

	if stage {
		args = append(args, "--stage")
	}

	for _, secret := range secrets {
		args = append(args, fmt.Sprintf("%s=%s", secret.Name, secret.Value))
	}

	_, err := util_cmd.
		NewCommandA("fly", args...).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithTimeout(30 * time.Second).
		WithTimeoutRetries(2).
		Run(ctx)

	if err != nil {
		return fmt.Errorf("error running 'fly secrets set' for app %s: %w", app, err)
	}

	return nil
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
		WithExtraArgs(accessTokenArgs(ctx)...).
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
		WithExtraArgs(accessTokenArgs(ctx)...).
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
	res, err := util_cmd.
		NewCommand("fly", "status", "-a", name).
		WithExtraArgs(accessTokenArgs(ctx)...).
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

	res, err := util_cmd.
		NewCommand("fly", "volumes", "list", "-a", name, "--json").
		WithExtraArgs(accessTokenArgs(ctx)...).
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

	res, err := util_cmd.
		NewCommand("fly", "config", "show", "-a", name).
		WithExtraArgs(accessTokenArgs(ctx)...).
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
	if !lo.Contains(allParams, "--region") && !lo.Contains(allParams, "-r") {
		allParams = append(allParams, "--region", cfg.PrimaryRegion)
	}
	_, err := tempDir.
		NewCommand("fly", allParams...).
		WithExtraArgs(accessTokenArgs(ctx)...).
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
	region string,
) error {
	allParams := append([]string{"deploy"}, cfg.DeployParams...)
	allParams = append(allParams, "--remote-only", "--detach")
	if !lo.Contains(allParams, "--region") && !lo.Contains(allParams, "-r") {
		allParams = append(allParams, "--region", region)
	}

	_, err := tempDir.
		NewCommand("fly", allParams...).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithTimeout(deployCfg.AttemptTimeout).
		WithTimeoutRetries(deployCfg.Retries).
		WithStdLogging().
		Run(ctx)
	if err != nil {
		return fmt.Errorf("error deploying app %s: %w", cfg.App, err)
	}
	return nil
}

func accessTokenArgs(ctx context.Context) []string {
	token := getAccessToken(ctx)
	if token == "" {
		return []string{}
	}
	return []string{"--access-token", token}
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
