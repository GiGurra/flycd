package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/flycd/util/util_git"
	"github.com/gigurra/flycd/internal/flycd/util/util_work_dir"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type TraverseAppTreeOptions struct {
	ValidAppCb       func(model.AppNode) error
	InvalidAppCb     func(model.AppNode) error
	ValidProjectCb   func(model.ProjectNode) error
	InvalidProjectCb func(model.ProjectNode) error
}

func TraverseDeepAppTree(
	ctx context.Context,
	path string,
	opts TraverseAppTreeOptions,
) error {

	analysis, err := scanDir(path)
	if err != nil {
		return fmt.Errorf("error analysing %s: %w", path, err)
	}

	apps, err := analysis.Apps()
	if err != nil {
		return fmt.Errorf("error getting apps at '%s': %w", path, err)
	}

	projects, err := analysis.Projects()
	if err != nil {
		return fmt.Errorf("error getting projects at '%s': %w", path, err)
	}

	for _, app := range apps {
		if app.IsValidApp() {
			if opts.ValidAppCb != nil {
				err := opts.ValidAppCb(app)
				if err != nil {
					return fmt.Errorf("error calling function for valid app %s @ %s: %w", app.AppConfig.App, app.Path, err)
				}
			}
		} else {
			if opts.InvalidAppCb != nil {
				err := opts.InvalidAppCb(app)
				if err != nil {
					return fmt.Errorf("error calling function for invalid app %s @ %s: %w", app.AppConfig.App, app.Path, err)
				}
			}
		}
	}

	for _, project := range projects {
		if project.IsValidProject() {

			if opts.ValidProjectCb != nil {
				err := opts.ValidProjectCb(project)
				if err != nil {
					return fmt.Errorf("error calling function for valid project %s @ %s: %w", project.ProjectConfig.Project, project.Path, err)
				}
			}

			switch project.ProjectConfig.Source.Type {
			case model.SourceTypeLocal:
				absPath := func() string {
					if filepath.IsAbs(project.ProjectConfig.Source.Path) {
						return project.ProjectConfig.Source.Path
					} else {
						return filepath.Join(project.Path, project.ProjectConfig.Source.Path)
					}
				}()
				err := TraverseDeepAppTree(ctx, absPath, opts)
				if err != nil {
					return fmt.Errorf("error traversing local project %s @ %s: %w", project.ProjectConfig.Project, project.Path, err)
				}
			case model.SourceTypeGit:

				err := func() error {

					// Clone to a temp folder
					tempDir, err := util_work_dir.NewTempDir("flycd-temp-cloned-project", "")
					if err != nil {
						return fmt.Errorf("creating temp dir for project %s: %w", project.ProjectConfig.Project, err)
					}
					defer tempDir.RemoveAll() // this is ok. We can wait until the end of the function

					// Clone to temp dir
					cloneResult, err := util_git.CloneShallow(ctx, project.ProjectConfig.Source, tempDir)
					if err != nil {
						return fmt.Errorf("cloning project %s: %w", project.ProjectConfig.Project, err)
					} else {
						err := TraverseDeepAppTree(ctx, filepath.Join(cloneResult.Dir.Cwd(), project.ProjectConfig.Source.Path), opts)
						if err != nil {
							return fmt.Errorf("error traversing cloned project %s @ %s: %w", project.ProjectConfig.Project, project.Path, err)
						}
					}

					return nil
				}()

				if err != nil {
					return err
				}

			default:
				fmt.Printf("unknown source type '%s' for project '%s' @ %s\n", project.ProjectConfig.Source.Type, project.ProjectConfig.Project, project.Path)
				if opts.InvalidProjectCb != nil {
					err := opts.InvalidProjectCb(project)
					if err != nil {
						return fmt.Errorf("error calling function for invalid project %s @ %s: %w", project.ProjectConfig.Project, project.Path, err)
					}
				}
			}

		} else {
			if opts.InvalidProjectCb != nil {
				err := opts.InvalidProjectCb(project)
				if err != nil {
					return fmt.Errorf("error calling function for invalid project %s @ %s: %w", project.ProjectConfig.Project, project.Path, err)
				}
			}
		}
	}

	return nil
}

func scanDir(path string) (model.SpecNode, error) {

	// convert path to absolut path
	path, err := filepath.Abs(path)
	if err != nil {
		return model.SpecNode{}, fmt.Errorf("error converting path to absolute path: %w", err)
	}

	nodeInfo, err := analyseTraversalCandidate(path)
	if err != nil {
		return model.SpecNode{}, fmt.Errorf("error analysing node '%s': %w", path, err)
	}

	if nodeInfo.HasAppYaml {
		workDir := util_work_dir.NewWorkDir(path)
		appYaml, err := workDir.ReadFile("app.yaml")
		if err != nil {
			return model.SpecNode{}, fmt.Errorf("error reading app.yaml: %w", err)
		}

		var appConfig model.AppConfig
		err = yaml.Unmarshal([]byte(appYaml), &appConfig)
		if err != nil {
			return model.SpecNode{
				Path: path,
				App: &model.AppNode{
					Path:               path,
					AppYaml:            appYaml,
					AppConfigSyntaxErr: err,
				},
				Children: []model.SpecNode{},
			}, nil
		}

		err = appConfig.Validate()
		if err != nil {
			return model.SpecNode{
				Path: path,
				App: &model.AppNode{
					Path:            path,
					AppYaml:         appYaml,
					AppConfigSemErr: err,
				},
				Children: []model.SpecNode{},
			}, nil
		}

		return model.SpecNode{
			Path: path,
			App: &model.AppNode{
				Path:      path,
				AppYaml:   appYaml,
				AppConfig: appConfig,
			},
			Children: []model.SpecNode{},
		}, nil
	} else if nodeInfo.HasProjectYaml {

		workDir := util_work_dir.NewWorkDir(path)
		projectYaml, err := workDir.ReadFile("project.yaml")
		if err != nil {
			return model.SpecNode{}, fmt.Errorf("error reading project.yaml: %w", err)
		}

		var projectConfig model.ProjectConfig
		err = yaml.Unmarshal([]byte(projectYaml), &projectConfig)
		if err != nil {
			return model.SpecNode{
				Path: path,
				Project: &model.ProjectNode{
					Path:                   path,
					ProjectYaml:            projectYaml,
					ProjectConfigSyntaxErr: err,
				},
				Children: []model.SpecNode{},
			}, nil
		}

		err = projectConfig.Validate()
		if err != nil {
			return model.SpecNode{
				Path: path,
				Project: &model.ProjectNode{
					Path:                path,
					ProjectYaml:         projectYaml,
					ProjectConfigSemErr: err,
				},
				Children: []model.SpecNode{},
			}, nil
		}

		return model.SpecNode{
			Path: path,
			Project: &model.ProjectNode{
				Path:          path,
				ProjectYaml:   projectYaml,
				ProjectConfig: projectConfig,
			},
			Children: []model.SpecNode{},
		}, nil
	} else if nodeInfo.HasProjectsDir {

		child, err := scanDir(filepath.Join(path, "projects"))
		if err != nil {
			return model.SpecNode{}, fmt.Errorf("error analysing children of node '%s': %w", path, err)
		}
		return model.SpecNode{
			Path:     path,
			Children: []model.SpecNode{child},
		}, nil
	} else {

		children := make([]model.SpecNode, len(nodeInfo.TraversableCandidates))
		for i, entry := range nodeInfo.TraversableCandidates {
			child, err := scanDir(filepath.Join(path, entry.Name()))
			if err != nil {
				return model.SpecNode{}, fmt.Errorf("error analysing children of node '%s': %w", path, err)
			}
			children[i] = child
		}

		return model.SpecNode{
			Path:     path,
			Children: children,
		}, nil
	}
}

func analyseTraversalCandidate(path string) (model.TraversalStepAnalysis, error) {

	entries, err := os.ReadDir(path)
	if err != nil {
		return model.TraversalStepAnalysis{}, fmt.Errorf("error reading directory: %w", err)
	}

	// Collect potentially traversable dirs
	traversableCandidates := make([]os.DirEntry, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			shouldTraverse, err := canTraverseDir(entry)
			if err != nil {
				return model.TraversalStepAnalysis{}, fmt.Errorf("error checking for symlink: %w", err)
			}
			if !shouldTraverse {
				continue
			}

			if entry.Name() == "projects" {
				return model.TraversalStepAnalysis{
					HasAppYaml:            false,
					HasProjectYaml:        false,
					HasProjectsDir:        true,
					TraversableCandidates: []os.DirEntry{},
				}, nil
			}

			traversableCandidates = append(traversableCandidates, entry)

		} else if entry.Name() == "app.yaml" {

			return model.TraversalStepAnalysis{
				HasAppYaml:            true,
				HasProjectYaml:        false,
				HasProjectsDir:        false,
				TraversableCandidates: []os.DirEntry{},
			}, nil

		} else if entry.Name() == "project.yaml" {

			return model.TraversalStepAnalysis{
				HasAppYaml:            false,
				HasProjectYaml:        true,
				HasProjectsDir:        false,
				TraversableCandidates: []os.DirEntry{},
			}, nil
		}
	}
	return model.TraversalStepAnalysis{
		HasAppYaml:            false,
		HasProjectYaml:        false,
		HasProjectsDir:        false,
		TraversableCandidates: traversableCandidates,
	}, nil
}

func isSymLink(dir os.DirEntry) (bool, error) {
	info, err := dir.Info()
	if err != nil {
		return false, fmt.Errorf("error getting symlink info: %w", err)
	}

	return info.Mode().Type() == os.ModeSymlink, nil
}

func canTraverseDir(dir os.DirEntry) (bool, error) {

	symlink, err := isSymLink(dir)
	if err != nil {
		return false, err
	}

	if symlink {
		return false, nil
	}

	if dir.Name() == ".git" || dir.Name() == ".actions" || dir.Name() == ".idea" || dir.Name() == ".vscode" {
		return false, nil
	}

	return true, nil
}
