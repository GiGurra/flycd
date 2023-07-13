package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"testing"
)

// Better start using a proper mock framework later :D

type DeployAppFromFolderInput struct {
	Path      string
	DeployCfg model.DeployConfig
}

type fakeDeployService struct {
	deployAppFromFolderInputs []DeployAppFromFolderInput
}

func newFakeDeployService() DeployService {
	return &fakeDeployService{
		deployAppFromFolderInputs: make([]DeployAppFromFolderInput, 0),
	}
}

func (f *fakeDeployService) DeployAll(
	_ context.Context,
	_ string,
	_ model.DeployConfig,
) (model.DeployResult, error) {
	// Not used by webhook service
	panic("won't be used")
}

func (f *fakeDeployService) DeployAppFromInlineConfig(
	_ context.Context,
	_ model.DeployConfig,
	_ model.AppConfig,
) (model.SingleAppDeploySuccessType, error) {
	// Not used by webhook service
	panic("won't be used")
}

func (f *fakeDeployService) DeployAppFromFolder(
	_ context.Context,
	path string,
	deployCfg model.DeployConfig,
) (model.SingleAppDeploySuccessType, error) {
	f.deployAppFromFolderInputs = append(f.deployAppFromFolderInputs, DeployAppFromFolderInput{
		Path:      path,
		DeployCfg: deployCfg,
	})
	return model.SingleAppDeployCreated, nil
}

var _ DeployService = &fakeDeployService{}

func TestNewWebHookServiceAbc(t *testing.T) {
	deployService := newFakeDeployService()
	webhookService := NewWebHookService(deployService)

	fmt.Printf("webhookService: %v\n", webhookService)
}
