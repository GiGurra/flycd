package flycd

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/internal/flycd/model"
	"github.com/gigurra/flycd/internal/flycd/util/util_git"
	"github.com/gigurra/flycd/internal/flycd/util/util_work_dir"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type TraversalStepAnalysis struct {
	HasAppYaml            bool
	HasProjectYaml        bool
	HasProjectsDir        bool
	TraversableCandidates []os.DirEntry
}

type AppNode struct {
	Path               string
	AppYaml            string
	AppConfig          model.AppConfig
	AppConfigSyntaxErr error
	AppConfigSemErr    error
}

type ProjectNode struct {
	Path                   string
	ProjectYaml            string
	ProjectConfig          model.ProjectConfig
	ProjectConfigSyntaxErr error
	ProjectConfigSemErr    error
}

type SpecNode struct {
	Path     string
	App      *AppNode
	Project  *ProjectNode
	Children []SpecNode
}

func (s SpecNode) Apps(followProjects ...bool) ([]AppNode, error) {

	follow := false
	if len(followProjects) > 0 {
		follow = followProjects[0]
	}

	nodeList, err := s.Flatten()
	if err != nil {
		return nil, fmt.Errorf("error flattening analysis: %w", err)
	}

	apps := lo.Filter(nodeList, func(node SpecNode, _ int) bool {
		return node.IsAppNode()
	})

	projects := lo.Filter(nodeList, func(node SpecNode, _ int) bool {
		return node.IsProjectNode()
	})

	if follow && len(projects) > 0 {
		fmt.Printf("analysis.SpecNode.Apps.follow: Not implemented yet!\n")
		fmt.Printf("Would have followed %d projects\n", len(projects))
		for _, project := range projects {
			fmt.Printf(" - %s\n", project.Path)
		}
	}

	return lo.Map(apps, func(item SpecNode, index int) AppNode {
		return *item.App
	}), nil
}

func (s SpecNode) Projects() ([]ProjectNode, error) {

	nodeList, err := s.Flatten()
	if err != nil {
		return nil, fmt.Errorf("error flattening analysis: %w", err)
	}

	projects := lo.Filter(nodeList, func(node SpecNode, _ int) bool {
		return node.IsProjectNode()
	})

	return lo.Map(projects, func(item SpecNode, index int) ProjectNode {
		return *item.Project
	}), nil
}

func (s SpecNode) Traverse(t func(node SpecNode) error) error {
	err := t(s)
	if err != nil {
		return fmt.Errorf("error traversing node '%s': %w", s.Path, err)
	}
	for _, child := range s.Children {
		err := child.Traverse(t)
		if err != nil {
			return fmt.Errorf("error traversing child node '%s': %w", child.Path, err)
		}
	}
	return nil
}

func (s SpecNode) Flatten() ([]SpecNode, error) {
	var result []SpecNode
	err := s.Traverse(func(node SpecNode) error {
		result = append(result, node)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error flattening node '%s': %w", s.Path, err)
	}
	return result, nil
}

func (s SpecNode) IsAppNode() bool {
	return s.App != nil && s.App.IsAppNode()
}

func (s SpecNode) IsProjectNode() bool {
	return s.Project != nil && s.Project.IsProjectNode()
}

func (s SpecNode) IsAppSyntaxValid() bool {
	return s.App != nil && s.App.IsAppSyntaxValid()
}

func (s SpecNode) IsValidApp() bool {
	return s.App != nil && s.App.IsValidApp()
}

func (s AppNode) IsAppNode() bool {
	return s.AppYaml != ""
}

func (s AppNode) IsAppSyntaxValid() bool {
	return s.IsAppNode() && s.AppConfig.App != "" && s.AppConfigSyntaxErr == nil
}

func (s AppNode) IsValidApp() bool {
	return s.IsAppNode() && s.IsAppSyntaxValid() && s.AppConfigSemErr == nil
}

func (s ProjectNode) IsProjectNode() bool {
	return s.ProjectYaml != ""
}

func (s ProjectNode) IsProjectSyntaxValid() bool {
	return s.IsProjectNode() && s.ProjectConfig.Project != "" && s.ProjectConfigSyntaxErr == nil
}

func (s ProjectNode) IsValidProject() bool {
	return s.IsProjectNode() && s.IsProjectSyntaxValid() && s.ProjectConfigSemErr == nil
}

type TraverseAppTreeOptions struct {
	ValidAppCb       func(AppNode) error
	InvalidAppCb     func(AppNode) error
	ValidProjectCb   func(ProjectNode) error
	InvalidProjectCb func(ProjectNode) error
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

func scanDir(path string) (SpecNode, error) {

	// convert path to absolut path
	path, err := filepath.Abs(path)
	if err != nil {
		return SpecNode{}, fmt.Errorf("error converting path to absolute path: %w", err)
	}

	nodeInfo, err := analyseTraversalCandidate(path)
	if err != nil {
		return SpecNode{}, fmt.Errorf("error analysing node '%s': %w", path, err)
	}

	if nodeInfo.HasAppYaml {
		workDir := util_work_dir.NewWorkDir(path)
		appYaml, err := workDir.ReadFile("app.yaml")
		if err != nil {
			return SpecNode{}, fmt.Errorf("error reading app.yaml: %w", err)
		}

		var appConfig model.AppConfig
		err = yaml.Unmarshal([]byte(appYaml), &appConfig)
		if err != nil {
			return SpecNode{
				Path: path,
				App: &AppNode{
					Path:               path,
					AppYaml:            appYaml,
					AppConfigSyntaxErr: err,
				},
				Children: []SpecNode{},
			}, nil
		}

		err = appConfig.Validate()
		if err != nil {
			return SpecNode{
				Path: path,
				App: &AppNode{
					Path:            path,
					AppYaml:         appYaml,
					AppConfigSemErr: err,
				},
				Children: []SpecNode{},
			}, nil
		}

		return SpecNode{
			Path: path,
			App: &AppNode{
				Path:      path,
				AppYaml:   appYaml,
				AppConfig: appConfig,
			},
			Children: []SpecNode{},
		}, nil
	} else if nodeInfo.HasProjectYaml {

		workDir := util_work_dir.NewWorkDir(path)
		projectYaml, err := workDir.ReadFile("project.yaml")
		if err != nil {
			return SpecNode{}, fmt.Errorf("error reading project.yaml: %w", err)
		}

		var projectConfig model.ProjectConfig
		err = yaml.Unmarshal([]byte(projectYaml), &projectConfig)
		if err != nil {
			return SpecNode{
				Path: path,
				Project: &ProjectNode{
					Path:                   path,
					ProjectYaml:            projectYaml,
					ProjectConfigSyntaxErr: err,
				},
				Children: []SpecNode{},
			}, nil
		}

		err = projectConfig.Validate()
		if err != nil {
			return SpecNode{
				Path: path,
				Project: &ProjectNode{
					Path:                path,
					ProjectYaml:         projectYaml,
					ProjectConfigSemErr: err,
				},
				Children: []SpecNode{},
			}, nil
		}

		return SpecNode{
			Path: path,
			Project: &ProjectNode{
				Path:          path,
				ProjectYaml:   projectYaml,
				ProjectConfig: projectConfig,
			},
			Children: []SpecNode{},
		}, nil
	} else if nodeInfo.HasProjectsDir {

		child, err := scanDir(filepath.Join(path, "projects"))
		if err != nil {
			return SpecNode{}, fmt.Errorf("error analysing children of node '%s': %w", path, err)
		}
		return SpecNode{
			Path:     path,
			Children: []SpecNode{child},
		}, nil
	} else {

		children := make([]SpecNode, len(nodeInfo.TraversableCandidates))
		for i, entry := range nodeInfo.TraversableCandidates {
			child, err := scanDir(filepath.Join(path, entry.Name()))
			if err != nil {
				return SpecNode{}, fmt.Errorf("error analysing children of node '%s': %w", path, err)
			}
			children[i] = child
		}

		return SpecNode{
			Path:     path,
			Children: children,
		}, nil
	}
}

func analyseTraversalCandidate(path string) (TraversalStepAnalysis, error) {

	entries, err := os.ReadDir(path)
	if err != nil {
		return TraversalStepAnalysis{}, fmt.Errorf("error reading directory: %w", err)
	}

	// Collect potentially traversable dirs
	traversableCandidates := make([]os.DirEntry, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			shouldTraverse, err := canTraverseDir(entry)
			if err != nil {
				return TraversalStepAnalysis{}, fmt.Errorf("error checking for symlink: %w", err)
			}
			if !shouldTraverse {
				continue
			}

			if entry.Name() == "projects" {
				return TraversalStepAnalysis{
					HasAppYaml:            false,
					HasProjectYaml:        false,
					HasProjectsDir:        true,
					TraversableCandidates: []os.DirEntry{},
				}, nil
			}

			traversableCandidates = append(traversableCandidates, entry)

		} else if entry.Name() == "app.yaml" {

			return TraversalStepAnalysis{
				HasAppYaml:            true,
				HasProjectYaml:        false,
				HasProjectsDir:        false,
				TraversableCandidates: []os.DirEntry{},
			}, nil

		} else if entry.Name() == "project.yaml" {

			return TraversalStepAnalysis{
				HasAppYaml:            false,
				HasProjectYaml:        true,
				HasProjectsDir:        false,
				TraversableCandidates: []os.DirEntry{},
			}, nil
		}
	}
	return TraversalStepAnalysis{
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
