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
	ctx := context.Background()
	go func() {

		err := TraverseDeepAppTree(ctx, path, TraverseAppTreeOptions{
			ValidAppCb: func(app AppNode) error {
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

					ctx := context.Background()
					deployCfg := model.
						NewDeployConfig().
						WithRetries(1).
						WithForce(false)
					_, err := DeploySingleAppFromFolder(ctx, app.Path, deployCfg)
					if err != nil {
						fmt.Printf("Error deploying app %s: %v\n", app.AppConfig.App, err)
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
