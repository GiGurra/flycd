package flycd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/flycd/util/util_cmd"
	"github.com/gigurra/flycd/internal/flycd/util/util_git"
	"github.com/gigurra/flycd/internal/flycd/util/util_work_dir"
	"path/filepath"
	"strings"
	"time"
)

type AppDeployFailure struct {
	Spec  AppNode
	Cause error
}

type ProjectDeployFailure struct {
	Spec  ProjectNode
	Cause error
}

type DeployResult struct {
	SucceededApps     []AppNode
	FailedApps        []AppDeployFailure
	SucceededProjects []AppNode
	FailedProjects    []ProjectDeployFailure
}

func (r DeployResult) Plus(other DeployResult) DeployResult {
	return DeployResult{
		SucceededApps:     append(r.SucceededApps, other.SucceededApps...),
		FailedApps:        append(r.FailedApps, other.FailedApps...),
		SucceededProjects: append(r.SucceededProjects, other.SucceededProjects...),
		FailedProjects:    append(r.FailedProjects, other.FailedProjects...),
	}
}

func (r DeployResult) Success() bool {
	return len(r.FailedApps) == 0 && len(r.FailedProjects) == 0
}

func (r DeployResult) HasErrors() bool {
	return len(r.FailedApps) != 0 || len(r.FailedProjects) != 0
}

func NewDeployResult() DeployResult {
	return DeployResult{
		SucceededApps:     make([]AppNode, 0),
		FailedApps:        make([]AppDeployFailure, 0),
		SucceededProjects: make([]AppNode, 0),
		FailedProjects:    make([]ProjectDeployFailure, 0),
	}
}

var SkippedNotValid = fmt.Errorf("skipped: not a valid app")
var SkippedAbortedEarlier = fmt.Errorf("skipped: job aborted earlier")

type FetchedProject struct {
	ProjectConfig model.ProjectConfig
	WorkDir       util_work_dir.WorkDir
	IsTempDir     bool
}

/*func FetchProject(config model.ProjectConfig) (FetchedProject, error) {
	switch config.Source.Type {
	case model.SourceTypeGit:
		return FetchProjectFromGit(config)
	case model.SourceTypeLocal:
		return FetchProjectFromLocal(config)
	default:
		return FetchedProject{}, fmt.Errorf("unsupported source type: %s", config.Source.Type)
	}
}*/

func DeployAll(
	ctx context.Context,
	path string,
	deployCfg DeployConfig,
) (DeployResult, error) {

	result := NewDeployResult()

	analysis, err := ScanDir(path)
	if err != nil {
		return NewDeployResult(), fmt.Errorf("error analysing %s: %w", path, err)
	}

	projects, err := analysis.Projects()
	if err != nil {
		return NewDeployResult(), fmt.Errorf("finding projects %s: %w", path, err)
	}

	apps, err := analysis.Apps()
	if err != nil {
		return NewDeployResult(), fmt.Errorf("finding apps %s: %w", path, err)
	}

	// Deploy projects
	for _, project := range projects {
		fmt.Printf("Considering project %s @ %s\n", project.ProjectConfig.Project, project.Path)
		if deployCfg.AbortOnFirstError && result.HasErrors() {
			fmt.Printf("Aborted earlier, skipping!\n")
			result.FailedProjects = append(result.FailedProjects, ProjectDeployFailure{
				Spec:  project,
				Cause: SkippedAbortedEarlier,
			})
			continue
		}
		if project.IsValidProject() {

			// Check what the project source is. Clone to a git repo if it is a git repo
			// Use local directly if it is a local folder
			switch project.ProjectConfig.Source.Type {
			case model.SourceTypeGit:

				err = func() error {

					// Clone to a temp folder
					tempDir, err := util_work_dir.NewTempDir("flycd-temp-cloned-project", "")
					if err != nil {
						return fmt.Errorf("creating temp dir for project %s: %w", project.ProjectConfig.Project, err)
					}
					defer tempDir.RemoveAll() // this is ok. We can wait until the end of the function

					// Clone to temp dir
					cloneResult, err := util_git.CloneShallow(ctx, project.ProjectConfig.Source, tempDir)
					if err != nil {
						result.FailedProjects = append(result.FailedProjects, ProjectDeployFailure{
							Spec:  project,
							Cause: fmt.Errorf("git clone failed for project %s: %w", project.ProjectConfig.Project, err),
						})
					} else {
						innerResult, err := DeployAll(ctx, cloneResult.Dir.Cwd(), deployCfg)
						if err != nil {
							return fmt.Errorf("deploying project %s: %w", project.ProjectConfig.Project, err)
						}

						result = result.Plus(innerResult)
					}

					return nil
				}()

				if err != nil {
					return result, err
				}

			case model.SourceTypeLocal:
				// Use the local folder directly
				absPath := func() string {
					if filepath.IsAbs(project.ProjectConfig.Source.Path) {
						return project.ProjectConfig.Source.Path
					} else {
						return filepath.Join(project.Path, project.ProjectConfig.Source.Path)
					}
				}()
				innerResult, err := DeployAll(ctx, absPath, deployCfg)
				if err != nil {
					return result, fmt.Errorf("deploying project %s: %w", project.ProjectConfig.Project, err)
				}

				result = result.Plus(innerResult)
			default:
				result.FailedProjects = append(result.FailedProjects, ProjectDeployFailure{
					Spec:  project,
					Cause: fmt.Errorf("unsupported source type: %s", project.ProjectConfig.Source.Type),
				})
			}
		} else {
			fmt.Printf("Project is NOT valid, skipping!\n")
			result.FailedProjects = append(result.FailedProjects, ProjectDeployFailure{
				Spec:  project,
				Cause: SkippedNotValid,
			})
		}
	}

	// Deploy apps
	for _, app := range apps {
		fmt.Printf("Considering app %s @ %s\n", app.AppConfig.App, app.Path)
		if deployCfg.AbortOnFirstError && result.HasErrors() {
			fmt.Printf("Aborted earlier, skipping!\n")
			result.FailedApps = append(result.FailedApps, AppDeployFailure{
				Spec:  app,
				Cause: SkippedAbortedEarlier,
			})
			continue
		}
		if app.IsValidApp() {
			err := DeploySingleAppFromFolder(ctx, app.Path, deployCfg)
			if err != nil {
				result.FailedApps = append(result.FailedApps, AppDeployFailure{
					Spec:  app,
					Cause: err,
				})
				fmt.Printf("Error deploying %s @ %s: %v\n:", app.AppConfig.App, app.Path, err)
			}
		} else {
			fmt.Printf("App is NOT valid, skipping!\n")
			result.FailedApps = append(result.FailedApps, AppDeployFailure{
				Spec:  app,
				Cause: SkippedNotValid,
			})
		}
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
