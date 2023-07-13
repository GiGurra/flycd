package model

import (
	"fmt"
	"github.com/samber/lo"
	"os"
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
	AppConfig          AppConfig
	AppConfigSyntaxErr error
	AppConfigSemErr    error
}

func (s AppNode) ErrCause() error {
	if s.AppConfigSemErr != nil {
		return s.AppConfigSemErr
	}
	if s.AppConfigSyntaxErr != nil {
		return s.AppConfigSyntaxErr
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

type SpecNode struct {
	Path     string
	App      *AppNode
	Project  *ProjectNode
	Children []SpecNode
}

func (s SpecNode) Apps() []AppNode {

	nodeList := s.Flatten()

	apps := lo.Filter(nodeList, func(node SpecNode, _ int) bool {
		return node.IsAppNode()
	})

	return lo.Map(apps, func(item SpecNode, index int) AppNode {
		return *item.App
	})
}

func (s SpecNode) Projects() []ProjectNode {

	nodeList := s.Flatten()
	projects := lo.Filter(nodeList, func(node SpecNode, _ int) bool {
		return node.IsProjectNode()
	})

	return lo.Map(projects, func(item SpecNode, index int) ProjectNode {
		return *item.Project
	})
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

func (s SpecNode) TraverseNoErr(t func(node SpecNode)) {
	t(s)
	for _, child := range s.Children {
		child.TraverseNoErr(t)
	}
}

func (s SpecNode) Flatten() []SpecNode {
	var result []SpecNode
	s.TraverseNoErr(func(node SpecNode) {
		result = append(result, node)
	})
	return result
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
