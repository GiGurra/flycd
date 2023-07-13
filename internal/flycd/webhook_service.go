package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/github"
	"github.com/samber/lo"
	"strings"
)

type WebHookService interface {
	HandleGithubWebhook(payload github.PushWebhookPayload, path string) <-chan error
}

type WebHookServiceImpl struct {
	deployService DeployService
}

func NewWebHookService(deployService DeployService) WebHookService {
	return &WebHookServiceImpl{
		deployService: deployService,
	}
}

func (w WebHookServiceImpl) HandleGithubWebhook(payload github.PushWebhookPayload, path string) <-chan error {

	ch := make(chan error, 1)

	// TODO: Implement some kind of persistence and/or caching here... So we don't have to clone everything every time...

	fmt.Printf("Traversing projects and apps for matching webhook url %s...\n", payload.Repository.Url)
	go func() {

		defer close(ch)

		matchedProjCount := 0
		ctx := context.Background()
		err := TraverseDeepAppTree(ctx, path, model.TraverseAppTreeOptions{
			ValidAppCb: func(app model.AppNode) error {

				if matchedProjCount > 0 || matchesApp(app, payload) {

					if matchedProjCount > 0 {
						fmt.Printf("App %s deploying because it is in a project that matches webhook %s...\n", app.AppConfig.App, payload.Repository.Url)
					} else {
						fmt.Printf("Found app %s matching webhook url %s. Deploying...\n", app.AppConfig.App, payload.Repository.Url)
					}

					deployCfg := model.
						NewDeployConfig().
						WithRetries(1).
						WithForce(false)
					_, err := w.deployService.DeployAppFromFolder(ctx, app.Path, deployCfg)
					if err != nil {
						fmt.Printf("Error deploying app %s: %v\n", app.AppConfig.App, err)
					}
				}
				return nil
			},
			BeginProjectCb: func(node model.ProjectNode) error {
				// unfortunately we have to deploy everything here :S. This is because we don't know how far down
				// the tree the change might affect our apps. So we have to deploy everything to be sure.
				// It would be better to just use app repo webhooks instead, or at least group apps into small projects

				if matchesProject(node, payload) {
					fmt.Printf("Found project %s matching webhook url %s. Deploying all apps in the project...\n", node.ProjectConfig.Project, payload.Repository.Url)
					matchedProjCount++
				}
				return nil
			},
			EndProjectCb: func(node model.ProjectNode) error {
				if matchesProject(node, payload) {
					matchedProjCount--
				}
				return nil
			},
		})
		if err != nil {
			fmt.Printf("error traversing app tree: %v", err)
			ch <- err
		}
	}()

	return ch
}

func matchesSpec(source model.Source, payload github.PushWebhookPayload) bool {
	if source.Repo == "" {
		return false
	}
	localKey := strings.ToLower(source.Repo)
	remoteKeys := allEntriesBothWithAndWithoutGitSuffix(lo.Uniq([]string{
		strings.ToLower(payload.Repository.Url),
		strings.ToLower(payload.Repository.CloneUrl),
		strings.ToLower(payload.Repository.HtmlUrl),
		strings.ToLower(payload.Repository.GitUrl),
		strings.ToLower(payload.Repository.SvnUrl),
		strings.ToLower(payload.Repository.SshUrl),
	}))
	return lo.Contains(remoteKeys, localKey)
}

func allEntriesBothWithAndWithoutGitSuffix(entries []string) []string {

	normalized := lo.Map(entries, func(entry string, _ int) string {
		return strings.TrimSuffix(entry, ".git")
	})

	withSuffix := lo.Map(normalized, func(entry string, _ int) string {
		return entry + ".git"
	})

	return lo.Uniq(append(normalized, withSuffix...))
}

func matchesApp(app model.AppNode, payload github.PushWebhookPayload) bool {
	return matchesSpec(app.AppConfig.Source, payload)
}

func matchesProject(project model.ProjectNode, payload github.PushWebhookPayload) bool {
	return matchesSpec(project.ProjectConfig.Source, payload)
}
