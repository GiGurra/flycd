package domain

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/mocks/domain"
	"github.com/gigurra/flycd/pkg/domain/model"
	"github.com/gigurra/flycd/pkg/ext/github"
	"github.com/stretchr/testify/mock"
	"path/filepath"
	"testing"
	"time"
)

func TestWebHookService(t *testing.T) {

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

			ctx, cancelFunc := context.WithCancel(context.Background())
			defer cancelFunc()

			fakeDeployService := domain.NewMockDeployService(t)
			webhookService := NewWebHookService(ctx, fakeDeployService)

			fmt.Printf("webhookService: %v\n", webhookService)

			payload := test.payload

			expPath, err := filepath.Abs("../../test/test-projects/webhooks/regular/app1")
			if err != nil {
				t.Fatalf("Failed to get abs path: %v", err)
			}

			fakeDeployService.
				EXPECT().
				DeployAppFromFolder(mock.Anything, expPath, mock.Anything, mock.Anything).
				Return(model.SingleAppDeployCreated, nil)

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
