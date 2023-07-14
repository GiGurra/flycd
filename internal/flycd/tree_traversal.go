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
	"strings"
)

type seen struct {
	Apps     map[string]bool
	Projects map[string]bool
}

func newSeen() seen {
	return seen{
		Apps:     make(map[string]bool),
		Projects: make(map[string]bool),
	}
}

func TraverseDeepAppTree(
	ctx context.Context,
	path string,
	opts model.TraverseAppTreeOptions,
) error {
	return doTraverseDeepAppTree(ctx, newSeen(), path, opts)
}

func doTraverseDeepAppTree(
	ctx context.Context,
	seen seen,
	path string,
	opts model.TraverseAppTreeOptions,
) error {

	analysis, err := scan(path)
	if err != nil {
		return fmt.Errorf("error analysing %s: %w", path, err)
	}

	apps := analysis.Apps()
	projects := analysis.Projects()

	// Must traverse projects before apps, to ensure desired project wrapping of apps in case of cyclic dependencies
	for _, project := range projects {

		if seen.Projects[project.ProjectConfig.Project] {
			if opts.SkippedProjectCb != nil {
				err := opts.SkippedProjectCb(project)
				if err != nil {
					return fmt.Errorf("error calling function for skipped project %s @ %s: %w", project.ProjectConfig.Project, project.Path, err)
				}
			}
			continue
		}

		seen.Projects[project.ProjectConfig.Project] = true

		if err := traverseProject(ctx, seen, opts, project); err != nil {
			return err
		}
	}

	for _, app := range apps {
		if app.IsValidApp() {

			if seen.Apps[app.AppConfig.App] {
				if opts.SkippedAppCb != nil {
					err := opts.SkippedAppCb(app)
					if err != nil {
						return fmt.Errorf("error calling function for skipped app %s @ %s: %w", app.AppConfig.App, app.Path, err)
					}
				}
				continue
			}

			seen.Apps[app.AppConfig.App] = true

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

	return nil
}

func traverseProject(
	ctx context.Context,
	seen seen,
	opts model.TraverseAppTreeOptions,
	project model.ProjectNode,
) error {
	if opts.BeginProjectCb != nil {
		err := opts.BeginProjectCb(project)
		if err != nil {
			return fmt.Errorf("error calling function for valid project %s @ %s: %w", project.ProjectConfig.Project, project.Path, err)
		}
	}

	defer func() {
		if opts.EndProjectCb != nil {
			err := opts.EndProjectCb(project)
			if err != nil {
				fmt.Printf("error calling function for valid project %s @ %s: %v", project.ProjectConfig.Project, project.Path, err)
			}
		}
	}()

	if project.IsValidProject() {

		switch project.ProjectConfig.Source.Type {
		case model.SourceTypeLocal:
			absPath := func() string {
				if filepath.IsAbs(project.ProjectConfig.Source.Path) {
					return project.ProjectConfig.Source.Path
				} else {
					return filepath.Join(project.Path, project.ProjectConfig.Source.Path)
				}
			}()
			err := doTraverseDeepAppTree(ctx, seen, absPath, opts)
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
					err := doTraverseDeepAppTree(ctx, seen, filepath.Join(cloneResult.Dir.Cwd(), project.ProjectConfig.Source.Path), opts)
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
			fmt.Printf("BUG: illegal or unknown source type '%s' for project '%s' @ %s\n", project.ProjectConfig.Source.Type, project.ProjectConfig.Project, project.Path)
		}

	}
	return nil
}

func scan(inputPath string) (model.SpecNode, error) {

	result := model.SpecNode{
		Path:     inputPath,
		Children: []model.SpecNode{},
	}

	// convert path to absolut path
	absPath, err := filepath.Abs(inputPath)
	if err != nil {
		return result, fmt.Errorf("error converting path to absolute path: %w", err)
	}

	nodeInfo, err := analyseTraversalCandidate(absPath)
	if err != nil {
		return result, fmt.Errorf("error analysing node '%s': %w", absPath, err)
	}
	path := nodeInfo.Path // Could be updated

	if nodeInfo.HasAppYaml {
		workDir := util_work_dir.NewWorkDir(path)
		appYaml, err := workDir.ReadFile("app.yaml")
		if err != nil {
			return result, fmt.Errorf("error reading app.yaml: %w", err)
		}

		var appConfig model.AppConfig
		err = yaml.Unmarshal([]byte(appYaml), &appConfig)
		if err != nil {
			result.App = &model.AppNode{
				Path:               path,
				AppYaml:            appYaml,
				AppConfigSyntaxErr: err,
			}
		} else {

			err = appConfig.Validate()
			if err != nil {
				result.App = &model.AppNode{
					Path:            path,
					AppYaml:         appYaml,
					AppConfigSemErr: err,
				}
			} else {

				result.App = &model.AppNode{
					Path:      path,
					AppYaml:   appYaml,
					AppConfig: appConfig,
				}
			}
		}
	}

	if nodeInfo.HasProjectYaml {

		workDir := util_work_dir.NewWorkDir(path)
		projectYaml, err := workDir.ReadFile("project.yaml")
		if err != nil {
			return result, fmt.Errorf("error reading project.yaml: %w", err)
		}

		var projectConfig model.ProjectConfig
		err = yaml.Unmarshal([]byte(projectYaml), &projectConfig)
		if err != nil {
			result.Project = &model.ProjectNode{
				Path:                   path,
				ProjectYaml:            projectYaml,
				ProjectConfigSyntaxErr: err,
			}
		} else {

			err = projectConfig.Validate()
			if err != nil {
				result.Project = &model.ProjectNode{
					Path:                path,
					ProjectYaml:         projectYaml,
					ProjectConfigSemErr: err,
				}
			} else {
				result.Project = &model.ProjectNode{
					Path:          path,
					ProjectYaml:   projectYaml,
					ProjectConfig: projectConfig,
				}
			}
		}
	}

	if nodeInfo.HasProjectsDir {

		child, err := scan(filepath.Join(path, "projects"))
		if err != nil {
			return result, fmt.Errorf("error analysing children of node '%s': %w", path, err)
		}

		result.Children = append(result.Children, child)
	} else {

		children := make([]model.SpecNode, len(nodeInfo.TraversableCandidates))
		for i, entry := range nodeInfo.TraversableCandidates {
			child, err := scan(filepath.Join(path, entry.Name()))
			if err != nil {
				return result, fmt.Errorf("error analysing children of node '%s': %w", path, err)
			}
			children[i] = child
		}

		result.Children = children
	}

	return result, nil
}

func analyseTraversalCandidate(path string) (model.TraversalStepAnalysis, error) {

	// check if path is file or dir
	fileInfo, err := os.Stat(path)
	if err != nil {
		return model.TraversalStepAnalysis{}, fmt.Errorf("error stating path '%s': %w", path, err)
	}

	if !fileInfo.IsDir() {
		if strings.HasSuffix(path, ".yaml") {
			dirPath := filepath.Dir(path)
			fileName := filepath.Base(path)
			if fileName == "app.yaml" {
				return model.TraversalStepAnalysis{
					Path:                  dirPath,
					HasAppYaml:            true,
					HasProjectYaml:        false,
					HasProjectsDir:        false,
					TraversableCandidates: []os.DirEntry{},
				}, nil
			} else if fileName == "project.yaml" {
				return model.TraversalStepAnalysis{
					Path:                  dirPath,
					HasAppYaml:            false,
					HasProjectYaml:        true,
					HasProjectsDir:        false,
					TraversableCandidates: []os.DirEntry{},
				}, nil
			} else {
				return model.TraversalStepAnalysis{}, fmt.Errorf("unexpected yaml file '%s'", path)
			}
		} else {
			return model.TraversalStepAnalysis{}, fmt.Errorf("unexpected file '%s'", path)
		}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return model.TraversalStepAnalysis{}, fmt.Errorf("error reading directory: %w", err)
	}

	// Collect potentially traversable dirs

	result := model.TraversalStepAnalysis{
		Path:                  path,
		HasAppYaml:            false,
		HasProjectYaml:        false,
		HasProjectsDir:        false,
		TraversableCandidates: []os.DirEntry{},
	}

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
				result.HasProjectsDir = true
			}

			result.TraversableCandidates = append(result.TraversableCandidates, entry)

		} else if entry.Name() == "app.yaml" {
			result.HasAppYaml = true

		} else if entry.Name() == "project.yaml" {
			result.HasProjectYaml = true
		}
	}

	if result.HasProjectsDir {
		// Here we don't want to look further down the tree in the regular way
		result.TraversableCandidates = []os.DirEntry{}
	}

	return result, nil
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