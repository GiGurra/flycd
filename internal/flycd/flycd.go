package flycd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/flycd/util/util_cmd"
	"github.com/gigurra/flycd/internal/flycd/util/util_work_dir"
	"strings"
	"time"
)

type AppDeployFailure struct {
	Spec  AppNode
	Cause error
}

type ProjectProcessingFailure struct {
	Spec  ProjectNode
	Cause error
}

type AppDeploySuccess struct {
	Spec        AppNode
	SuccessType SingleAppDeploySuccessType
}

type DeployResult struct {
	SucceededApps     []AppDeploySuccess
	FailedApps        []AppDeployFailure
	ProcessedProjects []ProjectNode
	FailedProjects    []ProjectProcessingFailure
}

func (r DeployResult) Plus(other DeployResult) DeployResult {
	return DeployResult{
		SucceededApps:     append(r.SucceededApps, other.SucceededApps...),
		FailedApps:        append(r.FailedApps, other.FailedApps...),
		ProcessedProjects: append(r.ProcessedProjects, other.ProcessedProjects...),
		FailedProjects:    append(r.FailedProjects, other.FailedProjects...),
	}
}

func (r DeployResult) Success() bool {
	return len(r.FailedApps) == 0 && len(r.FailedProjects) == 0
}

func (r DeployResult) HasErrors() bool {
	return len(r.FailedApps) != 0 || len(r.FailedProjects) != 0
}

func NewEmptyDeployResult() DeployResult {
	return DeployResult{
		SucceededApps:     make([]AppDeploySuccess, 0),
		FailedApps:        make([]AppDeployFailure, 0),
		ProcessedProjects: make([]ProjectNode, 0),
		FailedProjects:    make([]ProjectProcessingFailure, 0),
	}
}

var SkippedNotValid = fmt.Errorf("skipped: not a valid app")
var SkippedAbortedEarlier = fmt.Errorf("skipped: job aborted earlier")

type FetchedProject struct {
	ProjectConfig model.ProjectConfig
	WorkDir       util_work_dir.WorkDir
	IsTempDir     bool
}

func DeployAll(
	ctx context.Context,
	path string,
	deployCfg DeployConfig,
) (DeployResult, error) {

	result := NewEmptyDeployResult()

	err := TraverseDeepAppTree(ctx, path, TraverseAppTreeOptions{
		ValidAppCb: func(appNode AppNode) error {
			fmt.Printf("Considering app %s @ %s\n", appNode.AppConfig.App, appNode.Path)
			if deployCfg.AbortOnFirstError && result.HasErrors() {
				fmt.Printf("Aborted earlier, skipping!\n")
				result.FailedApps = append(result.FailedApps, AppDeployFailure{
					Spec:  appNode,
					Cause: SkippedAbortedEarlier,
				})
				return nil
			} else {
				res, err := DeploySingleAppFromFolder(ctx, appNode.Path, deployCfg)
				if err != nil {
					result.FailedApps = append(result.FailedApps, AppDeployFailure{
						Spec:  appNode,
						Cause: err,
					})
				} else {
					result.SucceededApps = append(result.SucceededApps, AppDeploySuccess{
						Spec:        appNode,
						SuccessType: res,
					})
				}
				return nil
			}
		},
		InvalidAppCb: func(appNode AppNode) error {
			result.FailedApps = append(result.FailedApps, AppDeployFailure{
				Spec:  appNode,
				Cause: SkippedNotValid,
			})
			return nil
		},
		ValidProjectCb: func(appNode ProjectNode) error {
			if deployCfg.AbortOnFirstError && result.HasErrors() {
				result.FailedProjects = append(result.FailedProjects, ProjectProcessingFailure{
					Spec:  appNode,
					Cause: SkippedAbortedEarlier,
				})
				return nil
			} else {
				result.ProcessedProjects = append(result.ProcessedProjects, appNode)
				return nil
			}
		},
		InvalidProjectCb: func(appNode ProjectNode) error {
			result.FailedProjects = append(result.FailedProjects, ProjectProcessingFailure{
				Spec:  appNode,
				Cause: SkippedNotValid,
			})
			return nil
		},
	})
	if err != nil {
		return result, fmt.Errorf("error traversing app tree: %w", err)
	}
	return result, nil
}

func ExistsApp(ctx context.Context, name string) (bool, error) {
	res, err := util_cmd.
		NewCommand("fly", "status", "-a", name).
		WithTimeout(10 * time.Second).
		WithTimeoutRetries(5).
		Run(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(res.Combined), "could not find app") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func GetDeployedAppConfig(ctx context.Context, name string) (model.AppConfig, error) {

	// Compare the current deployed appVersion with the new appVersion
	res, err := util_cmd.
		NewCommand("fly", "config", "show", "-a", name).
		WithTimeout(20 * time.Second).
		WithTimeoutRetries(5).
		Run(ctx)
	if err != nil {

		if strings.Contains(strings.ToLower(err.Error()), "no machines configured for this app") {
			return model.AppConfig{Env: map[string]string{}}, nil
		}

		return model.AppConfig{}, fmt.Errorf("error running fly config show for app %s: %w", name, err)
	}

	var deployedCfg model.AppConfig
	err = json.Unmarshal([]byte(res.StdOut), &deployedCfg)
	if err != nil {
		return model.AppConfig{}, fmt.Errorf("error unmarshalling fly config for app %s: %w", name, err)
	}

	return deployedCfg, nil
}
