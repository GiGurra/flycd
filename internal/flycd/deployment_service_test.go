package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/mocks"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestDeployFromFolder_newApp(t *testing.T) {
	ctx := context.Background()
	flyClient := mocks.NewMockFlyClient(t)
	deployService := NewDeployService(flyClient)
	deployCfg := model.
		NewDefaultDeployConfig().
		WithAbortOnFirstError(true).
		WithRetries(0)

	fmt.Printf("flyClient: %+v\n", flyClient)

	flyClient.
		EXPECT().
		ExistsApp(mock.Anything, mock.Anything).
		Return(false, nil)

	flyClient.
		EXPECT().
		CreateNewApp(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	flyClient.
		EXPECT().
		DeployExistingApp(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	_, err := deployService.DeployAppFromFolder(ctx, "../../test/test-projects/deploy-tests/apps/app1", deployCfg)
	if err != nil {
		t.Fatalf("DeployAppFromFolder failed: %v", err)
	}

}

func TestDeployFromFolder_existingApp(t *testing.T) {
	ctx := context.Background()
	flyClient := mocks.NewMockFlyClient(t)
	deployService := NewDeployService(flyClient)
	deployCfg := model.
		NewDefaultDeployConfig().
		WithAbortOnFirstError(true).
		WithRetries(0)

	fmt.Printf("flyClient: %+v\n", flyClient)

	flyClient.
		EXPECT().
		ExistsApp(mock.Anything, mock.Anything).
		Return(true, nil)

	flyClient.
		EXPECT().
		GetDeployedAppConfig(mock.Anything, mock.Anything).
		Return(model.AppConfig{}, nil)

	flyClient.
		EXPECT().
		DeployExistingApp(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	_, err := deployService.DeployAppFromFolder(ctx, "../../test/test-projects/deploy-tests/apps/app1", deployCfg)
	if err != nil {
		t.Fatalf("DeployAppFromFolder failed: %v", err)
	}

}

func TestDeployFromFolder_appMergingConfig(t *testing.T) {
	ctx := context.Background()
	flyClient := mocks.NewMockFlyClient(t)
	deployService := NewDeployService(flyClient)
	deployCfg := model.
		NewDefaultDeployConfig().
		WithAbortOnFirstError(true).
		WithRetries(0)

	fmt.Printf("flyClient: %+v\n", flyClient)

	flyClient.
		EXPECT().
		ExistsApp(mock.Anything, mock.Anything).
		Return(true, nil)

	flyClient.
		EXPECT().
		GetDeployedAppConfig(mock.Anything, mock.Anything).
		Return(model.AppConfig{}, nil)

	flyClient.
		EXPECT().
		DeployExistingApp(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	_, err := deployService.DeployAppFromFolder(ctx, "../../test/test-projects/deploy-tests/apps/app2", deployCfg)
	if err != nil {
		t.Fatalf("DeployAppFromFolder failed: %v", err)
	}

}
