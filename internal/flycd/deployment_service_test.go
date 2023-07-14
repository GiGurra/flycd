package flycd

import (
	"context"
	"github.com/gigurra/flycd/internal/fly_client"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/flycd/util/util_work_dir"
)

type fakeFlyClient struct {
}

func (f fakeFlyClient) CreateOrgToken(_ context.Context, _ string) (string, error) { panic("not used") }

func (f fakeFlyClient) ExistsSecret(ctx context.Context, cmd fly_client.ExistsSecretCmd) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (f fakeFlyClient) StoreSecret(ctx context.Context, cmd fly_client.StoreSecretCmd) error {
	//TODO implement me
	panic("implement me")
}

func (f fakeFlyClient) ExistsApp(ctx context.Context, name string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (f fakeFlyClient) GetDeployedAppConfig(ctx context.Context, name string) (model.AppConfig, error) {
	//TODO implement me
	panic("implement me")
}

func (f fakeFlyClient) CreateNewApp(ctx context.Context, cfg model.AppConfig, tempDir util_work_dir.WorkDir, twoStep bool) error {
	//TODO implement me
	panic("implement me")
}

func (f fakeFlyClient) DeployExistingApp(ctx context.Context, cfg model.AppConfig, tempDir util_work_dir.WorkDir, deployCfg model.DeployConfig) error {
	//TODO implement me
	panic("implement me")
}

var _ fly_client.FlyClient = &fakeFlyClient{}
