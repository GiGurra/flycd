package model

import "fmt"

type GitRef struct {
	Branch string `yaml:"branch,omitempty" toml:"branch"`
	Tag    string `yaml:"tag,omitempty" toml:"tag"`
	Commit string `yaml:"commit,omitempty" toml:"commit"`
}

func (g *GitRef) IsEmpty() bool {
	return g.Branch == "" && g.Tag == "" && g.Commit == ""
}

type Source struct {
	Repo   string     `yaml:"repo,omitempty" toml:"repo"`
	Path   string     `yaml:"path,omitempty" toml:"path"`
	Ref    GitRef     `yaml:"ref,omitempty" toml:"ref"`
	Type   SourceType `yaml:"type,omitempty" toml:"type"`
	Inline string     `yaml:"inline,omitempty" toml:"inline"`
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
	SourceTypeDocker           SourceType = "docker"
	SourceTypeInlineDockerFile SourceType = "inline-docker-file"
)

func (s *Source) Validate() error {

	if s == nil || *s == (Source{}) {
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
	case SourceTypeDocker:
		return fmt.Errorf("docker source type not implemented")
	default:
		return fmt.Errorf("invalid source type: '%s'", s.Type)
	}

	return nil
}
