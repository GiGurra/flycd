package domain

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/pkg/domain/model"
	"github.com/gigurra/flycd/pkg/ext/github"
	"github.com/samber/lo"
	"strings"
)

type WebHookService interface {
	HandleGithubWebhook(payload github.PushWebhookPayload, path string) <-chan error
	Start(ctx context.Context) error
	Stop()
}

type WebHookServiceImpl struct {
	deployService DeployService
	workQueue     chan func()
}

// Stop An alternative to cancelling the context itself
func (w *WebHookServiceImpl) Stop() {
	fmt.Printf("Closing webhook service\n")
	close(w.workQueue)
}

// Start Starts the internal worker
func (w *WebHookServiceImpl) Start(ctx context.Context) error {
	fmt.Printf("Creating webhook service & worker\n")

	// Prob add some way of preventing multiple workers from being started...

	go func() {
		for {
			select {
			case work, isOpen := <-w.workQueue:
				if isOpen {
					work()
				} else {
					fmt.Printf("Work queue closed: Stopping webhook worker\n")
					return
				}
			case <-ctx.Done():
				fmt.Printf("Context cancelled: Stopping webhook worker\n")
				return
			}
		}
	}()

	return nil
}

func NewWebHookService(deployService DeployService) WebHookService {
	return &WebHookServiceImpl{
		deployService: deployService,
		workQueue:     make(chan func(), 100),
	}
}

func (w WebHookServiceImpl) HandleGithubWebhook(payload github.PushWebhookPayload, path string) <-chan error {

	ch := make(chan error, 1)

	// TODO: Implement some kind of persistence and/or caching here... So we don't have to clone everything every time...

	task := func() {

		fmt.Printf("Start processing webhook %d for %s...\n", payload.HookId, payload.Repository.Url)

		defer close(ch)

		matchedProjCount := 0
		err := TraverseDeepAppTree(path, model.TraverseAppTreeContext{
			Context: context.Background(),
			ValidAppCb: func(ctx model.TraverseAppTreeContext, app model.AppAtFsNode) error {

				if matchedProjCount > 0 || matchesApp(app, payload) {

					if matchedProjCount > 0 {
						fmt.Printf("App %s deploying because it is in a project that matches webhook %s...\n", app.AppConfig.App, payload.Repository.Url)
					} else {
						fmt.Printf("Found app %s matching webhook url %s. Deploying...\n", app.AppConfig.App, payload.Repository.Url)
					}

					deployCfg := model.
						NewDefaultDeployConfig().
						WithRetries(1).
						WithForce(false)
					_, err := w.deployService.DeployAppFromFolder(ctx, app.Path, deployCfg, app.ToPreCalculatedApoConf())
					if err != nil {
						fmt.Printf("Error deploying app %s: %v\n", app.AppConfig.App, err)
					}
				}
				return nil
			},
			BeginProjectCb: func(ctx model.TraverseAppTreeContext, node model.ProjectAtFsNode) error {
				// unfortunately we have to deploy everything here :S. This is because we don't know how far down
				// the tree the change might affect our apps. So we have to deploy everything to be sure.
				// It would be better to just use app repo webhooks instead, or at least group apps into small projects

				if matchesProject(node, payload) {
					fmt.Printf("Found project %s matching webhook url %s. Deploying all apps in the project...\n", node.ProjectConfig.Project, payload.Repository.Url)
					matchedProjCount++
				}
				return nil
			},
			EndProjectCb: func(ctx model.TraverseAppTreeContext, node model.ProjectAtFsNode) error {
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

		fmt.Printf("Done processing webhook %d for %s...\n", payload.HookId, payload.Repository.Url)

	}

	w.workQueue <- task

	return ch
}

func matchesSpec(source model.Source, payload github.PushWebhookPayload) bool {
	if source.Repo == "" {
		return false
	}
	localKey := strings.ToLower(source.Repo)
	remoteKeys := allEntriesBothWithAndWithoutGitSuffix([]string{
		strings.ToLower(payload.Repository.Url),
		strings.ToLower(payload.Repository.CloneUrl),
		strings.ToLower(payload.Repository.HtmlUrl),
		strings.ToLower(payload.Repository.GitUrl),
		strings.ToLower(payload.Repository.SvnUrl),
		strings.ToLower(payload.Repository.SshUrl),
	})
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

func matchesApp(app model.AppAtFsNode, payload github.PushWebhookPayload) bool {
	return matchesSpec(app.AppConfig.Source, payload)
}

func matchesProject(project model.ProjectAtFsNode, payload github.PushWebhookPayload) bool {
	return matchesSpec(project.ProjectConfig.Source, payload)
}
