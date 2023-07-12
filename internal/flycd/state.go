package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/github"
	"github.com/samber/lo"
	"strings"
)

func CurrentApps(path string) ([]AppNode, error) {
	// TODO: Implement some caching here...
	return ScanForApps(path)
}

func CurrentProjects(path string) ([]ProjectNode, error) {
	// TODO: Implement some caching here...
	return ScanForProjects(path)
}

func HandleGithubWebhook(payload github.PushWebhookPayload, path string) error {

	// fin project that matches
	availabeProjects, err := CurrentProjects(path)
	if err != nil {
		return fmt.Errorf("error scanning for projects: %v", err)
	}

	fmt.Printf("Scanning %d projects for matching webhook url %s...\n", len(availabeProjects), payload.Repository.Url)
	for _, project := range availabeProjects {
		localKey := strings.ToLower(project.ProjectConfig.Source.Repo)
		remoteKeys := []string{
			strings.ToLower(payload.Repository.Url),
			strings.ToLower(payload.Repository.CloneUrl),
			strings.ToLower(payload.Repository.HtmlUrl),
			strings.ToLower(payload.Repository.GitUrl),
			strings.ToLower(payload.Repository.SvnUrl),
			strings.ToLower(payload.Repository.SshUrl),
		}
		if lo.Contains(remoteKeys, localKey) {
			fmt.Printf("Found project %s matching webhook url %s. Scheduling deploy...\n", project.ProjectConfig.Project, payload.Repository.Url)

			// TODO: Implement some kind of persistence here...
			fmt.Printf("Not implemented yet!\n")

			/*go func() {
				ctx := context.Background()
				deployCfg := NewDeployConfig().
					WithRetries(1).
					WithForce(false)
				_, err := DeployAll(ctx, app.Path, deployCfg)
				if err != nil {
					fmt.Printf("Error deploying app %s: %v\n", app.AppConfig.App, err)
				}
			}()*/
			return nil
		}
	}

	// find app that matches
	availableApps, err := CurrentApps(path)
	if err != nil {
		return fmt.Errorf("error scanning for apps: %v", err)
	}

	fmt.Printf("Scanning %d apps for matching webhook url %s...\n", len(availableApps), payload.Repository.Url)
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

			// TODO: Implement some kind of persistence here...

			go func() {
				ctx := context.Background()
				deployCfg := NewDeployConfig().
					WithRetries(1).
					WithForce(false)
				_, err := DeploySingleAppFromFolder(ctx, app.Path, deployCfg)
				if err != nil {
					fmt.Printf("Error deploying app %s: %v\n", app.AppConfig.App, err)
				}
			}()
			return nil
		}
	}

	fmt.Printf("WARNING: Could not find app or project matching webhook url: %s\n", payload.Repository.Url)

	return nil
}
