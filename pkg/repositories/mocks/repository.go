package mocks

import (
	"github.com/flyteorg/flyteadmin/pkg/repositories"
	"github.com/flyteorg/flyteadmin/pkg/repositories/interfaces"
)

type MockRepository struct {
	taskRepo               interfaces.TaskRepoInterface
	workflowRepo           interfaces.WorkflowRepoInterface
	launchPlanRepo         interfaces.LaunchPlanRepoInterface
	executionRepo          interfaces.ExecutionRepoInterface
	executionEventRepo     interfaces.ExecutionEventRepoInterface
	nodeExecutionRepo      interfaces.NodeExecutionRepoInterface
	nodeExecutionEventRepo interfaces.NodeExecutionEventRepoInterface
	projectRepo            interfaces.ProjectRepoInterface
	resourceRepo           interfaces.ResourceRepoInterface
	taskExecutionRepo      interfaces.TaskExecutionRepoInterface
	namedEntityRepo        interfaces.NamedEntityRepoInterface
}

func (r *MockRepository) TaskRepo() interfaces.TaskRepoInterface {
	return r.taskRepo
}

func (r *MockRepository) WorkflowRepo() interfaces.WorkflowRepoInterface {
	return r.workflowRepo
}

func (r *MockRepository) LaunchPlanRepo() interfaces.LaunchPlanRepoInterface {
	return r.launchPlanRepo
}

func (r *MockRepository) ExecutionRepo() interfaces.ExecutionRepoInterface {
	return r.executionRepo
}

func (r *MockRepository) ExecutionEventRepo() interfaces.ExecutionEventRepoInterface {
	return r.executionEventRepo
}

func (r *MockRepository) NodeExecutionRepo() interfaces.NodeExecutionRepoInterface {
	return r.nodeExecutionRepo
}

func (r *MockRepository) NodeExecutionEventRepo() interfaces.NodeExecutionEventRepoInterface {
	return r.nodeExecutionEventRepo
}

func (r *MockRepository) ProjectRepo() interfaces.ProjectRepoInterface {
	return r.projectRepo
}

func (r *MockRepository) ResourceRepo() interfaces.ResourceRepoInterface {
	return r.resourceRepo
}

func (r *MockRepository) TaskExecutionRepo() interfaces.TaskExecutionRepoInterface {
	return r.taskExecutionRepo
}

func (r *MockRepository) NamedEntityRepo() interfaces.NamedEntityRepoInterface {
	return r.namedEntityRepo
}

func NewMockRepository() repositories.RepositoryInterface {
	return &MockRepository{
		taskRepo:               NewMockTaskRepo(),
		workflowRepo:           NewMockWorkflowRepo(),
		launchPlanRepo:         NewMockLaunchPlanRepo(),
		executionRepo:          NewMockExecutionRepo(),
		nodeExecutionRepo:      NewMockNodeExecutionRepo(),
		projectRepo:            NewMockProjectRepo(),
		resourceRepo:           NewMockResourceRepo(),
		taskExecutionRepo:      NewMockTaskExecutionRepo(),
		namedEntityRepo:        NewMockNamedEntityRepo(),
		executionEventRepo:     &ExecutionEventRepoInterface{},
		nodeExecutionEventRepo: &NodeExecutionEventRepoInterface{},
	}
}
