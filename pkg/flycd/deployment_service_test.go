package flycd

import (
	"context"
	"fmt"
	mocks "github.com/gigurra/flycd/mocks/pkg/fly_client"
	"github.com/gigurra/flycd/pkg/flycd/model"
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
		DeployExistingApp(mock.Anything, mock.Anything, mock.Anything, mock.Anything, "arn").
		Return(nil)

	_, err := deployService.DeployAppFromFolder(ctx, "../../test/test-projects/deploy-tests/apps/app1", deployCfg, nil)
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
		DeployExistingApp(mock.Anything, mock.Anything, mock.Anything, mock.Anything, "arn").
		Return(nil)

	_, err := deployService.DeployAppFromFolder(ctx, "../../test/test-projects/deploy-tests/apps/app1", deployCfg, nil)
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
		DeployExistingApp(mock.Anything, mock.Anything, mock.Anything, mock.Anything, "arn").
		Return(nil)

	_, err := deployService.DeployAppFromFolder(ctx, "../../test/test-projects/deploy-tests/apps/app2", deployCfg, nil)
	if err != nil {
		t.Fatalf("DeployAppFromFolder failed: %v", err)
	}

}

func TestDeployFromFolder_withVolumes(t *testing.T) {

	for _, test := range []struct {
		name               string
		deployCfg          model.DeployConfig
		numDeployedVolumes int
		region             string
		deployedAppScale   int
		numExtendedVolumes int
		numCreatedVolumes  int
	}{
		{
			name: "create volumes - current app scale decides",
			deployCfg: model.
				NewDefaultDeployConfig().
				WithAbortOnFirstError(true).
				WithRetries(0),
			numDeployedVolumes: 0,
			deployedAppScale:   4,
			numExtendedVolumes: 0,
			numCreatedVolumes:  4,
			region:             "arn",
		},
		{
			name: "create volumes - minimum services count decides",
			deployCfg: model.
				NewDefaultDeployConfig().
				WithAbortOnFirstError(true).
				WithRetries(0),
			numDeployedVolumes: 0,
			deployedAppScale:   0,
			numExtendedVolumes: 0,
			numCreatedVolumes:  3,
			region:             "arn",
		},
		{
			name: "resize and create volumes",
			deployCfg: model.
				NewDefaultDeployConfig().
				WithAbortOnFirstError(true).
				WithRetries(0),
			numDeployedVolumes: 2,
			deployedAppScale:   0,
			numExtendedVolumes: 2,
			numCreatedVolumes:  1,
			region:             "arn",
		},
		{
			name: "create volumes when existing ones are in the wrong region",
			deployCfg: model.
				NewDefaultDeployConfig().
				WithAbortOnFirstError(true).
				WithRetries(0),
			numDeployedVolumes: 2,
			deployedAppScale:   0,
			numExtendedVolumes: 2,
			numCreatedVolumes:  1,
			region:             "arn",
		},
	} {

		t.Run(test.name, func(t *testing.T) {

			ctx := context.Background()
			flyClient := mocks.NewMockFlyClient(t)
			deployService := NewDeployService(flyClient)

			fmt.Printf("flyClient: %+v\n", flyClient)

			flyClient.
				EXPECT().
				ExistsApp(mock.Anything, mock.Anything).
				Return(true, nil)

			flyClient.
				EXPECT().
				GetDeployedAppConfig(mock.Anything, mock.Anything).
				Return(model.AppConfig{}, nil)

			alreadyDeployedVolumes := []model.VolumeState{}
			for i := 0; i < test.numDeployedVolumes; i++ {
				alreadyDeployedVolumes = append(alreadyDeployedVolumes, model.VolumeState{
					ID:     fmt.Sprintf("volume-%d", i),
					Name:   "data",
					SizeGb: 9,
					Region: test.region,
				})
			}
			flyClient.
				EXPECT().
				GetAppVolumes(mock.Anything, mock.Anything).
				Return(alreadyDeployedVolumes, nil)

			flyClient.
				EXPECT().
				GetAppScale(mock.Anything, mock.Anything).
				Return([]model.ScaleState{
					{
						Process: "app",
						Count:   test.deployedAppScale,
						Regions: map[string]int{test.region: test.deployedAppScale},
					},
				}, nil)

			if test.numCreatedVolumes > 0 {
				flyClient.
					EXPECT().
					CreateVolume(mock.Anything, "nginx-with-volumes-test", model.VolumeConfig{
						Name:   "data",
						SizeGb: 10,
					}, "arn").
					Return(model.VolumeState{}, nil).
					Times(test.numCreatedVolumes)
			}

			if test.numExtendedVolumes > 0 {
				flyClient.
					EXPECT().
					ExtendVolume(mock.Anything, "nginx-with-volumes-test", mock.Anything, 10).
					Return(nil).
					Times(test.numExtendedVolumes)
			}

			flyClient.
				EXPECT().
				DeployExistingApp(mock.Anything, mock.Anything, mock.Anything, mock.Anything, "arn").
				Return(nil)

			_, err := deployService.DeployAppFromFolder(ctx, "../../test/test-projects/nginx-with-volumes/app", test.deployCfg, nil)
			if err != nil {
				t.Fatalf("DeployAppFromFolder failed: %v", err)
			}
		})
	}

}
