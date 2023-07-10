package flycd

import (
	"fmt"
	"os"
)

type SingleFolderAnalysisResult struct {
	HasAppYaml            bool
	HasProjectsDir        bool
	TraversableCandidates []os.DirEntry
}

func analyseSingleFolder(path string) (SingleFolderAnalysisResult, error) {

	entries, err := os.ReadDir(path)
	if err != nil {
		return SingleFolderAnalysisResult{}, fmt.Errorf("error reading directory: %w", err)
	}

	// Collect potentially traversable dirs
	traversableCandidates := make([]os.DirEntry, 0)
	hasAppYaml := false
	hasProjectsDir := false

	for _, entry := range entries {
		if entry.IsDir() {
			shouldTraverse, err := canTraverseDir(entry)
			if err != nil {
				return SingleFolderAnalysisResult{}, fmt.Errorf("error checking for symlink: %w", err)
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
	return SingleFolderAnalysisResult{
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
