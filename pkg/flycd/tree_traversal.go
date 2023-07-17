package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/pkg/flycd/model"
	"github.com/gigurra/flycd/pkg/flycd/util/util_git"
	"github.com/gigurra/flycd/pkg/flycd/util/util_work_dir"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
)

func TraverseDeepAppTree(
	path string,
	ctx model.TraverseAppTreeContext,
) error {
	if ctx.Context == nil {
		ctx.Context = context.Background()
	}
	if ctx.Seen.Apps == nil {
		ctx.Seen.Apps = map[string]bool{}
	}
	if ctx.Seen.Projects == nil {
		ctx.Seen.Projects = map[string]bool{}
	}
	return doTraverseDeepAppTree(path, ctx)
}

func doTraverseDeepAppTree(
	path string,
	ctx model.TraverseAppTreeContext,
) error {

	analysis, err := scanFsNode(ctx, path)
	if err != nil {
		return fmt.Errorf("error analysing %s: %w", path, err)
	}

	apps := analysis.Apps()
	projects := analysis.Projects()

	// Must traverse projects before apps, to ensure desired project wrapping of apps in case of cyclic dependencies
	for _, project := range projects {

		if ctx.Seen.Projects[project.ProjectConfig.Project] {
			if ctx.SkippedProjectCb != nil {
				err := ctx.SkippedProjectCb(ctx, project)
				if err != nil {
					return fmt.Errorf("error calling function for skipped project %s @ %s: %w", project.ProjectConfig.Project, project.Path, err)
				}
			}
			continue
		}

		ctx.Seen.Projects[project.ProjectConfig.Project] = true

		if err := traverseProject(ctx, project); err != nil {
			return err
		}
	}

	for _, app := range apps {
		if app.IsValidApp() {

			if ctx.Seen.Apps[app.AppConfig.App] {
				if ctx.SkippedAppCb != nil {
					err := ctx.SkippedAppCb(ctx, app)
					if err != nil {
						return fmt.Errorf("error calling function for skipped app %s @ %s: %w", app.AppConfig.App, app.Path, err)
					}
				}
				continue
			}

			ctx.Seen.Apps[app.AppConfig.App] = true

			if ctx.ValidAppCb != nil {
				err := ctx.ValidAppCb(ctx, app)
				if err != nil {
					return fmt.Errorf("error calling function for valid app %s @ %s: %w", app.AppConfig.App, app.Path, err)
				}
			}
		} else {
			if ctx.InvalidAppCb != nil {
				err := ctx.InvalidAppCb(ctx, app)
				if err != nil {
					return fmt.Errorf("error calling function for invalid app %s @ %s: %w", app.AppConfig.App, app.Path, err)
				}
			}
		}
	}

	return nil
}

func calcCommonAppCfg(projectConfigs []model.ProjectConfig) (model.CommonAppConfig, error) {
	commonAppCfg := model.CommonAppConfig{}
	for _, projectCfg := range projectConfigs {
		err := projectCfg.Validate()
		if err != nil {
			return commonAppCfg, fmt.Errorf("error validating project config %s: %w", projectCfg.Project, err)
		} else {
			commonAppCfg = commonAppCfg.Plus(projectCfg.Common)
		}
	}
	return commonAppCfg, nil
}

func traverseProject(
	ctx model.TraverseAppTreeContext,
	project model.ProjectAtFsNode,
) error {
	if ctx.BeginProjectCb != nil {
		err := ctx.BeginProjectCb(ctx, project)
		if err != nil {
			return fmt.Errorf("error calling function for valid project %s @ %s: %w", project.ProjectConfig.Project, project.Path, err)
		}
	}

	ctx.Parents = append(ctx.Parents, project.ProjectConfig)
	commonAppCfgBefore := ctx.CommonAppCfg
	CommonAppCfgAfter, err := calcCommonAppCfg(ctx.Parents)
	if err != nil {
		return fmt.Errorf("error calculating common app config for project %s @ %s: %w", project.ProjectConfig.Project, project.Path, err)
	}
	ctx.CommonAppCfg = CommonAppCfgAfter
	defer func() {
		ctx.CommonAppCfg = commonAppCfgBefore
		ctx.Parents = ctx.Parents[:len(ctx.Parents)-1]
	}()

	defer func() {
		if ctx.EndProjectCb != nil {
			err := ctx.EndProjectCb(ctx, project)
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
			err := doTraverseDeepAppTree(absPath, ctx)
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
					err := doTraverseDeepAppTree(filepath.Join(cloneResult.Dir.Cwd(), project.ProjectConfig.Source.Path), ctx)
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

func scanFsNode(
	ctx model.TraverseAppTreeContext,
	inputPath string,
) (model.FsNode, error) {

	result := model.FsNode{
		Path:     inputPath,
		Children: []model.FsNode{},
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

		cfgTyped, cfgUntyped, errCfg :=
			ctx.CommonAppCfg.MakeAppConfig([]byte(appYaml))

		result.App = &model.AppAtFsNode{
			Path:             path,
			AppYaml:          appYaml,
			AppConfigUntyped: cfgUntyped,
			AppConfig:        cfgTyped,
			AppConfigErr:     errCfg,
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
			result.Project = &model.ProjectAtFsNode{
				Path:                   path,
				ProjectYaml:            projectYaml,
				ProjectConfigSyntaxErr: err,
			}
		} else {

			err = projectConfig.Validate()
			if err != nil {
				result.Project = &model.ProjectAtFsNode{
					Path:                path,
					ProjectYaml:         projectYaml,
					ProjectConfigSemErr: err,
				}
			} else {
				result.Project = &model.ProjectAtFsNode{
					Path:          path,
					ProjectYaml:   projectYaml,
					ProjectConfig: projectConfig,
				}
			}
		}
	}

	if nodeInfo.HasProjectsDir {

		child, err := scanFsNode(ctx, filepath.Join(path, "projects"))
		if err != nil {
			return result, fmt.Errorf("error analysing children of node '%s': %w", path, err)
		}

		result.Children = append(result.Children, child)
	} else {

		children := make([]model.FsNode, len(nodeInfo.TraversableCandidates))
		for i, entry := range nodeInfo.TraversableCandidates {
			child, err := scanFsNode(ctx, filepath.Join(path, entry.Name()))
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
