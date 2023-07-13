package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/github"
	"path/filepath"
	"testing"
)

// Better start using a proper mock framework later :D

type DeployAppFromFolderInput struct {
	Path      string
	DeployCfg model.DeployConfig
}

type fakeDeployServiceT struct {
	deployAppFromFolderInputs []DeployAppFromFolderInput
}

func newFakeDeployService() *fakeDeployServiceT {
	return &fakeDeployServiceT{
		deployAppFromFolderInputs: make([]DeployAppFromFolderInput, 0),
	}
}

func (f *fakeDeployServiceT) DeployAll(
	_ context.Context,
	_ string,
	_ model.DeployConfig,
) (model.DeployResult, error) {
	// Not used by webhook service
	panic("won't be used")
}

func (f *fakeDeployServiceT) DeployAppFromInlineConfig(
	_ context.Context,
	_ model.DeployConfig,
	_ model.AppConfig,
) (model.SingleAppDeploySuccessType, error) {
	// Not used by webhook service
	panic("won't be used")
}

func (f *fakeDeployServiceT) DeployAppFromFolder(
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

var _ DeployService = &fakeDeployServiceT{}

func TestNewWebHookServiceAbc(t *testing.T) {
	fakeDeployService := newFakeDeployService()
	webhookService := NewWebHookService(fakeDeployService)

	fmt.Printf("webhookService: %v\n", webhookService)

	payload := generateTestPushWebhookPayload()

	ch := webhookService.HandleGithubWebhook(payload, "../../test/test-projects/webhooks/regular")

	select {
	case err, open := <-ch:
		if err != nil {
			t.Fatalf("Failed to handle webhook: %v", err)
		}
		if open {
			t.Fatalf("Expected channel to be closed")
		}
	}

	if len(fakeDeployService.deployAppFromFolderInputs) != 1 {
		t.Fatalf("Expected 1 deployAppFromFolder call, got %d", len(fakeDeployService.deployAppFromFolderInputs))
	}

	input := fakeDeployService.deployAppFromFolderInputs[0]
	expPath, err := filepath.Abs("../../test/test-projects/webhooks/regular/app1")
	if err != nil {
		t.Fatalf("Failed to get abs path: %v", err)
	}
	if input.Path != expPath {
		t.Fatalf("Expected path to be '%s', got '%s'", expPath, input.Path)
	}
}

func generateTestPushWebhookPayload() github.PushWebhookPayload {
	// Create test User
	testUser := github.User{
		Name:  "Test User",
		Email: "testuser@example.com",
	}

	// Create test Repository
	testRepo := github.Repository{
		ID:            123,
		Name:          "Test Repo",
		FullName:      "Test User/Test Repo",
		Private:       false,
		HtmlUrl:       "https://github.com/TestUser/TestRepo",
		Url:           "https://github.com/TestUser/TestRepo",
		CreatedAt:     1616161616,
		UpdatedAt:     "2022-01-01T00:00:00Z",
		PushedAt:      1616161616,
		GitUrl:        "git://github.com/TestUser/TestRepo.git",
		SshUrl:        "git@github.com:TestUser/TestRepo.git",
		CloneUrl:      "https://github.com/TestUser/TestRepo.git",
		SvnUrl:        "https://svn.github.com/TestUser/TestRepo",
		Visibility:    "public",
		DefaultBranch: "main",
		MasterBranch:  "main",
	}

	// Create test Commit
	testCommit := github.Commit{
		ID:        "abc123",
		TreeID:    "def456",
		Message:   "Test commit",
		Timestamp: "2022-01-01T00:00:00Z",
		URL:       "https://github.com/TestUser/TestRepo/commit/abc123",
		Author:    testUser,
	}

	// Create test PushWebhookPayload
	testPayload := github.PushWebhookPayload{
		Ref:        "refs/heads/main",
		Repository: testRepo,
		Pusher:     testUser,
		HeadCommit: testCommit,
	}

	return testPayload
}
