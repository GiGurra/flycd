package model

import (
	"context"
	"fmt"
	"github.com/samber/lo"
	"os"
)

type Seen struct {
	Apps     map[string]bool
	Projects map[string]bool
}

func NewSeen() Seen {
	return Seen{
		Apps:     make(map[string]bool),
		Projects: make(map[string]bool),
	}
}

type TraverseAppTreeContext struct {
	context.Context
	ValidAppCb       func(TraverseAppTreeContext, AppNode) error
	InvalidAppCb     func(TraverseAppTreeContext, AppNode) error
	SkippedAppCb     func(TraverseAppTreeContext, AppNode) error
	BeginProjectCb   func(TraverseAppTreeContext, ProjectNode) error
	EndProjectCb     func(TraverseAppTreeContext, ProjectNode) error
	SkippedProjectCb func(TraverseAppTreeContext, ProjectNode) error
	Seen             Seen
	Parents          []ProjectConfig
	CommonAppCfg     CommonAppConfig
}

// provet that TraverseAppTreeContext implements the context interface
var _ context.Context = TraverseAppTreeContext{}

type TraversalStepAnalysis struct {
	Path                  string
	HasAppYaml            bool
	HasProjectYaml        bool
	HasProjectsDir        bool
	TraversableCandidates []os.DirEntry
}

type AppNode struct {
	Path             string
	AppYaml          string
	AppConfigUntyped map[string]any
	AppConfig        AppConfig
	AppConfigErr     error
}

func (s AppNode) ToPreCalculatedApoConf() *PreCalculatedAppConfig {
	return &PreCalculatedAppConfig{
		Typed:   s.AppConfig,
		UnTyped: s.AppConfigUntyped,
	}
}

func (s AppNode) ErrCause() error {
	if s.AppConfigErr != nil {
		return s.AppConfigErr
	}
	return nil
}

type ProjectNode struct {
	Path                   string
	ProjectYaml            string
	ProjectConfig          ProjectConfig
	ProjectConfigSyntaxErr error
	ProjectConfigSemErr    error
}

func (s ProjectNode) ErrCause() error {
	if s.ProjectConfigSemErr != nil {
		return s.ProjectConfigSemErr
	}
	if s.ProjectConfigSyntaxErr != nil {
		return s.ProjectConfigSyntaxErr
	}
	return nil
}

type FsNode struct {
	Path     string
	App      *AppNode
	Project  *ProjectNode
	Children []FsNode
}

func (s FsNode) Apps() []AppNode {

	nodeList := s.Flatten()

	apps := lo.Filter(nodeList, func(node FsNode, _ int) bool {
		return node.HasAppNode()
	})

	return lo.Map(apps, func(item FsNode, index int) AppNode {
		return *item.App
	})
}

func (s FsNode) Projects() []ProjectNode {

	nodeList := s.Flatten()
	projects := lo.Filter(nodeList, func(node FsNode, _ int) bool {
		return node.HasProjectNode()
	})

	return lo.Map(projects, func(item FsNode, index int) ProjectNode {
		return *item.Project
	})
}

func (s FsNode) Traverse(t func(node FsNode) error) error {
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

func (s FsNode) TraverseNoErr(t func(node FsNode)) {
	t(s)
	for _, child := range s.Children {
		child.TraverseNoErr(t)
	}
}

func (s FsNode) Flatten() []FsNode {
	var result []FsNode
	s.TraverseNoErr(func(node FsNode) {
		result = append(result, node)
	})
	return result
}

func (s FsNode) HasAppNode() bool {
	return s.App != nil && s.App.IsAppNode()
}

func (s FsNode) HasProjectNode() bool {
	return s.Project != nil && s.Project.IsProjectNode()
}

func (s FsNode) IsAppSyntaxValid() bool {
	return s.App != nil && s.App.IsAppSyntaxValid()
}

func (s FsNode) IsValidApp() bool {
	return s.App != nil && s.App.IsValidApp()
}

func (s AppNode) IsAppNode() bool {
	return s.AppYaml != ""
}

func (s AppNode) IsAppSyntaxValid() bool {
	return s.IsAppNode() && s.AppConfig.App != ""
}

func (s AppNode) IsValidApp() bool {
	return s.IsAppNode() && s.IsAppSyntaxValid() && s.AppConfigErr == nil
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
