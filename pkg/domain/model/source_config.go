package model

import (
	"fmt"
	"github.com/gigurra/flycd/pkg/util/util_git"
	"reflect"
)

type GitRef struct {
	Branch string `yaml:"branch,omitempty" toml:"branch" json:"branch,omitempty"`
	Tag    string `yaml:"tag,omitempty" toml:"tag" json:"tag,omitempty"`
	Commit string `yaml:"commit,omitempty" toml:"commit" json:"commit,omitempty"`
}

func (g *GitRef) IsEmpty() bool {
	return g.Branch == "" && g.Tag == "" && g.Commit == ""
}

type MergeCfg struct {
	All     bool     `yaml:"all,omitempty" toml:"all" json:"all,omitempty"`
	Include []string `yaml:"include,omitempty" toml:"include,omitempty" json:"include,omitempty"`
	// TODO: Implement
	//Exclude []string `yaml:"exclude,omitempty" toml:"exclude,omitempty" json:"exclude,omitempty"`
}

type Source struct {
	Repo   string     `yaml:"repo,omitempty" toml:"repo" json:"repo,omitempty"`
	Path   string     `yaml:"path,omitempty" toml:"path" json:"path,omitempty"`
	Ref    GitRef     `yaml:"ref,omitempty" toml:"ref" json:"ref,omitempty"`
	Type   SourceType `yaml:"type,omitempty" toml:"type" json:"type,omitempty"`
	Inline string     `yaml:"inline,omitempty" toml:"inline" json:"inline,omitempty"`
}

func (s *Source) AsGitCloneSource() util_git.CloneSource {
	return util_git.CloneSource{
		Repo:   s.Repo,
		Branch: s.Ref.Branch,
		Tag:    s.Ref.Tag,
		Commit: s.Ref.Commit,
	}
}

func NewInlineDockerFileSource(inline string) Source {
	return Source{
		Type:   SourceTypeInlineDockerFile,
		Inline: inline,
	}
}

func NewLocalFolderSource(path string) Source {
	return Source{
		Type: SourceTypeLocal,
		Path: path,
	}
}

type SourceType string

const (
	SourceTypeGit              SourceType = "git"
	SourceTypeLocal            SourceType = "local"
	SourceTypeInlineDockerFile SourceType = "inline-docker-file"

	// Not much point in implementing SourceTypeDocker before fly.io supports private registries
	// SourceTypeDocker           SourceType = "docker"
)

func (s *Source) Validate() error {

	if s == nil || reflect.DeepEqual(*s, Source{}) {
		return fmt.Errorf(".source is required")
	}

	switch s.Type {
	case SourceTypeGit:
		if s.Repo == "" {
			return fmt.Errorf("repo is required")
		}
	case SourceTypeLocal:
	case SourceTypeInlineDockerFile:
		if s.Inline == "" {
			return fmt.Errorf("inline docker file is required")
		}
	default:
		return fmt.Errorf("invalid source type: '%s'", s.Type)
	}

	return nil
}
