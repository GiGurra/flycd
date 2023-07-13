package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/github"
	"github.com/samber/lo"
	"strings"
)

func HandleGithubWebhook(payload github.PushWebhookPayload, path string) error {

	// TODO: Implement some kind of persistence and/or caching here... So we don't have to clone everything every time...

	fmt.Printf("Traversing projects and apps for matching webhook url %s...\n", payload.Repository.Url)
	go func() {

		ctx := context.Background()
		err := TraverseDeepAppTree(ctx, path, TraverseAppTreeOptions{
			ValidAppCb: func(app model.AppNode) error {
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
					fmt.Printf("Found app %s matching webhook url %s. Deploying...\n", app.AppConfig.App, payload.Repository.Url)

					deployCfg := model.
						NewDeployConfig().
						WithRetries(1).
						WithForce(false)
					_, err := DeployAppFromFolder(ctx, app.Path, deployCfg)
					if err != nil {
						fmt.Printf("Error deploying app %s: %v\n", app.AppConfig.App, err)
					}
				}
				return nil
			},
			ValidProjectCb: func(node model.ProjectNode) error {
				// unfortunately we have to deploy everything here :S. This is because we don't know how far down
				// the tree the change might affect our apps. So we have to deploy everything to be sure.
				// It would be better to just use app repo webhooks instead, or at least group apps into small projects
				localKey := strings.ToLower(node.ProjectConfig.Source.Repo)
				remoteKeys := []string{
					strings.ToLower(payload.Repository.Url),
					strings.ToLower(payload.Repository.CloneUrl),
					strings.ToLower(payload.Repository.HtmlUrl),
					strings.ToLower(payload.Repository.GitUrl),
					strings.ToLower(payload.Repository.SvnUrl),
					strings.ToLower(payload.Repository.SshUrl),
				}

				if lo.Contains(remoteKeys, localKey) {
					fmt.Printf("Found project %s matching webhook url %s. Deploying all apps in the project...\n", node.ProjectConfig.Project, payload.Repository.Url)

					deployCfg := model.
						NewDeployConfig().
						WithRetries(1).
						WithForce(false)
					_, err := DeployAll(ctx, node.Path, deployCfg)
					if err != nil {
						fmt.Printf("Error deploying project %s: %v\n", node.ProjectConfig.Project, err)
					}
				}
				return nil
			},
		})
		if err != nil {
			fmt.Printf("error traversing app tree: %v", err)
		}
	}()

	return nil
}
