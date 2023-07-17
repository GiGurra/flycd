package util_git

import (
	"context"
	"fmt"
	"github.com/gigurra/flycd/pkg/util/util_work_dir"
	"strings"
)

type GitCloneResult struct {
	Dir  util_work_dir.WorkDir
	Hash string
}

type CloneSource struct {
	Repo   string
	Branch string
	Tag    string
	Commit string
}

func CloneShallow(
	ctx context.Context,
	source CloneSource,
	workDir util_work_dir.WorkDir,
) (GitCloneResult, error) {

	var err error

	if source.Commit != "" {
		// Shallow clone of specific commit
		// https://stackoverflow.com/questions/31278902/how-to-shallow-clone-a-specific-commit-with-depth-1
		_, err = workDir.
			NewCommand("git", "init").
			WithStdLogging().
			Run(ctx)
		if err != nil {
			return GitCloneResult{}, fmt.Errorf("error initializing git repo: %w", err)
		}

		_, err = workDir.
			NewCommand("git", "remote", "add", "origin", source.Repo).
			WithStdLogging().
			Run(ctx)
		if err != nil {
			return GitCloneResult{}, fmt.Errorf("error adding git remote: %w", err)
		}

		_, err = workDir.
			NewCommand("git", "fetch", "--depth", "1", "origin", source.Commit).
			WithStdLogging().
			Run(ctx)
		if err != nil {
			return GitCloneResult{}, fmt.Errorf("error fetching git commit: %w", err)
		}

		_, err = workDir.
			NewCommand("git", "checkout", "FETCH_HEAD").
			WithStdLogging().
			Run(ctx)
		if err != nil {
			return GitCloneResult{}, fmt.Errorf("error checking out git commit: %w", err)
		}

	} else if source.Tag != "" {
		_, err = workDir.
			NewCommand("git", "clone", source.Repo, "repo", "--depth", "1", "--branch", source.Tag).
			WithStdLogging().
			Run(ctx)
		if err != nil {
			return GitCloneResult{}, fmt.Errorf("error cloning git repo %s: %w", source.Repo, err)
		}
		workDir = workDir.WithChildCwd("repo")

	} else if source.Branch != "" {
		_, err = workDir.
			NewCommand("git", "clone", source.Repo, "repo", "--depth", "1", "--branch", source.Branch).
			WithStdLogging().
			Run(ctx)
		if err != nil {
			return GitCloneResult{}, fmt.Errorf("error cloning git repo %s: %w", source.Repo, err)
		}
		workDir = workDir.WithChildCwd("repo")
	} else {
		_, err = workDir.NewCommand("git", "clone", source.Repo, "repo", "--depth", "1").
			WithStdLogging().
			Run(ctx)
		if err != nil {
			return GitCloneResult{}, fmt.Errorf("error cloning git repo %s: %w", source.Repo, err)
		}
		workDir = workDir.WithChildCwd("repo")
	}

	res, err := workDir.
		NewCommand("git", "rev-parse", "HEAD").
		Run(ctx)
	if err != nil {
		return GitCloneResult{}, fmt.Errorf("error getting git commit hash: %w", err)
	}

	return GitCloneResult{
		Dir:  workDir,
		Hash: strings.TrimSpace(res.StdOut),
	}, nil
}
