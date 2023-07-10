package flycd

import (
	"flycd/internal/flycd/util/util_work_dir"
	"fmt"
	"os"
	"path/filepath"
)

type TraversalStepAnalysis struct {
	HasAppYaml            bool
	HasProjectsDir        bool
	TraversableCandidates []os.DirEntry
}

type SpecNode struct {
	Path      string
	AppYaml   string
	AppConfig AppConfig
	Children  []SpecNode
}

func (s SpecNode) IsAppNode() bool {
	return s.AppYaml != ""
}

func (s SpecNode) IsAppSyntaxValid() bool {
	return s.AppConfig.App != "" // it has been parsed
}

func (s SpecNode) IsAppValid() bool {
	return s.IsAppNode() && s.IsAppSyntaxValid() && s.AppConfig.Validate() == nil
}

func AnalyseSpec(path string) (SpecNode, error) {

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
		return SpecNode{
			Path:     path,
			AppYaml:  appYaml,
			Children: []SpecNode{},
		}, nil
	} else if nodeInfo.HasProjectsDir {

		child, err := AnalyseSpec(filepath.Join(path, "projects"))
		if err != nil {
			return SpecNode{}, fmt.Errorf("error analysing children of node '%s': %w", path, err)
		}
		return SpecNode{
			Path:     path,
			AppYaml:  "",
			Children: []SpecNode{child},
		}, nil
	} else {

		children := make([]SpecNode, len(nodeInfo.TraversableCandidates))
		for i, entry := range nodeInfo.TraversableCandidates {
			child, err := AnalyseSpec(filepath.Join(path, entry.Name()))
			if err != nil {
				return SpecNode{}, fmt.Errorf("error analysing children of node '%s': %w", path, err)
			}
			children[i] = child
		}

		return SpecNode{
			Path:     path,
			AppYaml:  "",
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
	hasAppYaml := false
	hasProjectsDir := false

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
				hasProjectsDir = true
			}

			traversableCandidates = append(traversableCandidates, entry)

		} else if entry.Name() == "app.yaml" {

			hasAppYaml = true

		}
	}
	return TraversalStepAnalysis{
		HasAppYaml:            hasAppYaml,
		HasProjectsDir:        hasProjectsDir,
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
