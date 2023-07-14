package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/github"
	"path/filepath"
	"testing"
	"time"
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

func TestWebHookService(t *testing.T) {

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	Init(ctx)

	for _, test := range []struct {
		name    string
		path    string
		payload github.PushWebhookPayload
	}{
		{
			name:    "regular payload",
			path:    "../../test/test-projects/webhooks/regular",
			payload: generateTestPushWebhookPayload(),
		},
		{
			name:    "payload missing .git in urls",
			path:    "../../test/test-projects/webhooks/regular",
			payload: generateTestPushWebhookPayloadWithoutGitSuffix(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {

			fakeDeployService := newFakeDeployService()
			webhookService := NewWebHookService(fakeDeployService)

			fmt.Printf("webhookService: %v\n", webhookService)

			payload := test.payload

			ch := webhookService.HandleGithubWebhook(payload, "../../test/test-projects/webhooks/regular")

			select {
			case err, open := <-ch:
				if err != nil {
					t.Fatalf("Failed to handle webhook: %v", err)
				}
				if open {
					t.Fatalf("Expected channel to be closed")
				}
			case <-time.After(5 * time.Second):
				t.Fatalf("Timed out waiting for webhook to be handled")
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
		})
	}

}

func generateTestPushWebhookPayloadWithoutGitSuffix() github.PushWebhookPayload {
	result := generateTestPushWebhookPayload()
	result.Repository.GitUrl = "git://github.com/TestUser/TestRepo"
	result.Repository.SshUrl = "git@github.com:TestUser/TestRepo"
	return result
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
		ID:      "abc123",
		TreeID:  "def456",
		Message: "Test commit",
		URL:     "https://github.com/TestUser/TestRepo/commit/abc123",
		Author:  testUser,
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
