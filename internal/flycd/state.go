package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/github"
	"github.com/samber/lo"
	"strings"
)

func CurrentApps(path string) ([]SpecNode, error) {
	// TODO: Implement some caching here...
	return ScanForApps(path)
}

func HandleGithubWebhook(payload github.PushWebhookPayload, path string) error {
	availableApps, err := CurrentApps(path)
	if err != nil {
		return fmt.Errorf("error scanning for apps: %v", err)
	}

	// find app that matches
	for _, app := range availableApps {
		localKey := strings.ToLower(app.AppConfig.Source.Repo)
		remoteKeys := []string{
			strings.ToLower(payload.Repository.Url),
			strings.ToLower(payload.Repository.CloneUrl),
			strings.ToLower(payload.Repository.HtmlUrl),
			strings.ToLower(payload.Repository.GitUrl),
			strings.ToLower(payload.Repository.SvnUrl),
			strings.ToLower(payload.Repository.SshUrl),
		}
		if lo.Contains(remoteKeys, localKey) {
			fmt.Printf("Found app %s matching webhook url %s. Scheduling deploy...\n", app.AppConfig.App, payload.Repository.Url)
			go func() {
				ctx := context.Background()
				deployCfg := NewDeployConfig().
					WithRetries(1).
					WithForce(false)
				err := DeploySingleAppFromFolder(ctx, app.Path, deployCfg)
				if err != nil {
					fmt.Printf("Error deploying app %s: %v\n", app.AppConfig.App, err)
				}
			}()
			return nil
		}
	}

	fmt.Printf("WARNING: Could not find app matching webhook url: %s\n", payload.Repository.Url)

	return nil
}