package fly_client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/GiGurra/cmder"
	"github.com/gigurra/flycd/pkg/domain/model"
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

	ScaleAppRam(
		ctx context.Context,
		app string,
		ramMb int,
	) error

	ScaleAppVm(
		ctx context.Context,
		app string,
		vm string,
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

	if ip.V == model.IpV4 {
		params = append(params, "--yes")
	}

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

	res := cmder.
		New(params...).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(1 * time.Minute).
		WithRetries(1).
		Run(ctx)

	if res.Err != nil {
		return fmt.Errorf("error allocating ip %+v for app %s: %w", ip, app, res.Err)
	} else {
		return nil
	}
}

func (c FlyClientImpl) DeleteIp(
	ctx context.Context,
	app string,
	_ string,
	address string,
) error {

	res := cmder.
		New("fly", "ips", "release", address, "-a", app).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(1 * time.Minute).
		WithRetries(1).
		Run(ctx)

	if res.Err != nil {
		return fmt.Errorf("error releasing ip %s for app %s: %w", address, app, res.Err)
	} else {
		return nil
	}
}

func (c FlyClientImpl) ListIps(ctx context.Context, app string) ([]IpListItem, error) {

	res := cmder.
		New("fly", "ips", "list", "-a", app, "--json").
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(1 * time.Minute).
		WithRetries(1).
		Run(ctx)
	if res.Err != nil {
		return nil, fmt.Errorf("error getting ips list. Do you have a token loaded?: %w", res.Err)
	}

	// Prob chang this to use json instead
	items := make([]IpListItem, 0)
	err := json.Unmarshal([]byte(res.StdOut), &items)
	if err != nil {
		return nil, fmt.Errorf("error parsing ips list: %w", err)
	}

	return items, nil
}

func (c FlyClientImpl) ListApps(ctx context.Context) ([]AppListItem, error) {

	// ensure we have a token loaded for the org we are monitoring
	res := cmder.
		New("fly", "apps", "list").
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(2 * time.Minute).
		WithRetries(1).
		Run(ctx)
	if res.Err != nil {
		return nil, fmt.Errorf("error getting apps list. Do you have a token loaded?: %w", res.Err)
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

	res := cmder.
		NewA("fly", "tokens", "create", "org", orgSlug).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(10 * time.Second).
		WithRetries(1).
		Run(ctx)

	if res.Err != nil {
		return "", fmt.Errorf("error running 'fly tokens create org': %w", res.Err)
	}

	// Run the command
	stdOut := res.StdOut

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

	res := cmder.
		NewA("fly", "scale", "show", "-a", app, "--json").
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(20 * time.Second).
		WithRetries(2).
		Run(ctx)

	if res.Err != nil {
		return nil, fmt.Errorf("error running 'fly scale show -a %s --json': %w", app, res.Err)
	}

	var scaleStates []model.ScaleState
	err := json.Unmarshal([]byte(res.StdOut), &scaleStates)
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

	res := cmder.
		NewA("fly", "scale", "count", strconv.FormatInt(int64(count), 10), "--app", app, "--region", region, "-y").
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(120 * time.Second).
		WithRetries(1).
		Run(ctx)

	if res.Err != nil {
		return fmt.Errorf("error running 'fly scale count %d --app %s --region %s -y': %w", count, app, region, res.Err)
	}

	return nil
}

func (c FlyClientImpl) ScaleAppRam(
	ctx context.Context,
	app string,
	ramMb int,
) error {

	res := cmder.
		NewA("fly", "scale", "memory", fmt.Sprintf("%d", ramMb), "--app", app).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(360 * time.Second).
		WithRetries(1).
		Run(ctx)

	if res.Err != nil {
		return fmt.Errorf("error running 'fly scale memory %d --app %s -y': %w", ramMb, app, res.Err)
	}

	return nil
}

func (c FlyClientImpl) ScaleAppVm(
	ctx context.Context,
	app string,
	vm string,
) error {

	res := cmder.
		NewA("fly", "scale", "vm", vm, "--app", app).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(360 * time.Second).
		WithRetries(1).
		Run(ctx)

	if res.Err != nil {
		return fmt.Errorf("error running 'fly scale vm %s --app %s -y': %w", vm, app, res.Err)
	}

	return nil
}

func (c FlyClientImpl) ExtendVolume(
	ctx context.Context,
	appName string,
	volumeId string,
	gb int,
) error {

	res := cmder.
		NewA("fly", "volume", "extend", volumeId, "-a", appName, "-s", strconv.FormatInt(int64(gb), 10)).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(60 * time.Second).
		WithRetries(2).
		Run(ctx)

	if res.Err != nil {
		return fmt.Errorf("error running 'fly volume extend %s -a %s': %w", volumeId, appName, res.Err)
	}

	return nil
}

func (c FlyClientImpl) CreateVolume(
	ctx context.Context,
	app string,
	cfg model.VolumeConfig,
	region string,
) (model.VolumeState, error) {

	res := cmder.
		NewA("fly", "volumes", "create", cfg.Name, "--region", region, "-s", strconv.FormatInt(int64(cfg.SizeGb), 10), "--app", app, "-y", "--json").
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(60 * time.Second).
		WithRetries(0).
		Run(ctx)

	if res.Err != nil {
		return model.VolumeState{}, fmt.Errorf("error running 'fly volumes create' for app %s: %w", app, res.Err)
	}

	var volumeState model.VolumeState
	err := json.Unmarshal([]byte(res.StdOut), &volumeState)
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

	res := cmder.
		NewA("fly", args...).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(30 * time.Second).
		WithRetries(2).
		Run(ctx)

	if res.Err != nil {
		return fmt.Errorf("error running 'fly secrets set' for app %s: %w", app, res.Err)
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

	res := cmder.
		NewA("fly", args...).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(10 * time.Second).
		WithRetries(5).
		Run(ctx)
	if res.Err != nil {
		return false, fmt.Errorf("error running fly secrets list for '%s': %w", cmd.AppName, res.Err)
	}

	// Parse strResp as json array of flySecretListItem
	var secrets []flySecretListItem
	err := json.Unmarshal([]byte(res.StdOut), &secrets)
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

	res := cmder.
		NewA("fly", args...).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(240 * time.Second).
		WithRetries(5).
		WithStdOutErrForwarded().
		Run(ctx)
	if res.Err != nil {
		return fmt.Errorf("error running fly secrets set for '%s': %w", cmd.SecretName, res.Err)
	}

	return nil
}

func (c FlyClientImpl) ExistsApp(ctx context.Context, name string) (bool, error) {
	res := cmder.
		New("fly", "status", "-a", name).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(10 * time.Second).
		WithRetries(5).
		Run(ctx)
	if res.Err != nil {
		if strings.Contains(strings.ToLower(res.Combined), "could not find app") {
			return false, nil
		}
		return false, res.Err
	}
	return true, nil
}

func (c FlyClientImpl) GetAppVolumes(
	ctx context.Context,
	name string,
) ([]model.VolumeState, error) {

	res := cmder.
		New("fly", "volumes", "list", "-a", name, "--json").
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(20 * time.Second).
		WithRetries(5).
		Run(ctx)

	if res.Err != nil {
		return []model.VolumeState{}, fmt.Errorf("error running fly volumes list for app %s: %w", name, res.Err)
	}

	var volumes []model.VolumeState
	err := json.Unmarshal([]byte(res.StdOut), &volumes)
	if err != nil {
		return []model.VolumeState{}, fmt.Errorf("error parsing fly volumes list for app %s: %w", name, err)
	}

	return volumes, nil
}

func (c FlyClientImpl) GetDeployedAppConfig(ctx context.Context, name string) (model.AppConfig, error) {

	res := cmder.
		New("fly", "config", "show", "-a", name).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(20 * time.Second).
		WithRetries(5).
		Run(ctx)
	if res.Err != nil {

		if strings.Contains(strings.ToLower(res.Err.Error()), "no machines configured for this app") {
			return model.AppConfig{Env: map[string]string{}}, nil
		}

		return model.AppConfig{}, fmt.Errorf("error running fly config show for app %s: %w", name, res.Err)
	}

	var deployedCfg model.AppConfig
	err := json.Unmarshal([]byte(res.StdOut), &deployedCfg)
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
	res := tempDir.
		NewCommand("fly", allParams...).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(20 * time.Second).
		WithRetries(5).
		WithStdOutErrForwarded().
		Run(ctx)
	if res.Err != nil {
		return fmt.Errorf("error creating app %s: %w", cfg.App, res.Err)
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

	res := tempDir.
		NewCommand("fly", allParams...).
		WithExtraArgs(accessTokenArgs(ctx)...).
		WithAttemptTimeout(deployCfg.AttemptTimeout).
		WithRetries(deployCfg.Retries).
		WithStdOutErrForwarded().
		Run(ctx)
	if res.Err != nil {
		return fmt.Errorf("error deploying app %s: %w", cfg.App, res.Err)
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
