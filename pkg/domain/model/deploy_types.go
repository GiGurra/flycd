package model

type SingleAppDeploySuccessType string

const (
	SingleAppDeployCreated  SingleAppDeploySuccessType = "created"
	SingleAppDeployUpdated  SingleAppDeploySuccessType = "updated"
	SingleAppDeployNoChange SingleAppDeploySuccessType = "no-change"
)

type AppDeployFailure struct {
	Spec  AppAtFsNode
	Cause error
}

type ProjectProcessingFailure struct {
	Spec  ProjectAtFsNode
	Cause error
}

type AppDeploySuccess struct {
	Spec        AppAtFsNode
	SuccessType SingleAppDeploySuccessType
}

type DeployResult struct {
	SucceededApps     []AppDeploySuccess
	FailedApps        []AppDeployFailure
	ProcessedProjects []ProjectAtFsNode
	FailedProjects    []ProjectProcessingFailure
}

func (r DeployResult) Plus(other DeployResult) DeployResult {
	return DeployResult{
		SucceededApps:     append(r.SucceededApps, other.SucceededApps...),
		FailedApps:        append(r.FailedApps, other.FailedApps...),
		ProcessedProjects: append(r.ProcessedProjects, other.ProcessedProjects...),
		FailedProjects:    append(r.FailedProjects, other.FailedProjects...),
	}
}

func (r DeployResult) Success() bool {
	return len(r.FailedApps) == 0 && len(r.FailedProjects) == 0
}

func (r DeployResult) HasErrors() bool {
	return len(r.FailedApps) != 0 || len(r.FailedProjects) != 0
}

func NewEmptyDeployResult() DeployResult {
	return DeployResult{
		SucceededApps:     make([]AppDeploySuccess, 0),
		FailedApps:        make([]AppDeployFailure, 0),
		ProcessedProjects: make([]ProjectAtFsNode, 0),
		FailedProjects:    make([]ProjectProcessingFailure, 0),
	}
}
