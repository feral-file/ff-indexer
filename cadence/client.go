package cadence

import (
	"context"

	"github.com/spf13/viper"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/workflow"
)

var CadenceService = "cadence-frontend"

// WorkerClient manages multiple cadence worker service clients
type WorkerClient struct {
	Domain string

	clients map[string]client.Client
}

func NewWorkerClient(domain string) *WorkerClient {
	return &WorkerClient{
		Domain:  domain,
		clients: map[string]client.Client{},
	}
}

// AddService register a service client
func (c *WorkerClient) AddService(clientName string) {
	serviceClient := BuildCadenceServiceClient(viper.GetString("cadence.host_port"), clientName, CadenceService)

	cadenceWorker := client.NewClient(
		serviceClient,
		c.Domain,
		&client.Options{},
	)

	c.clients[clientName] = cadenceWorker
}

// StartWorkflow triggers a workflow in a specific client
func (c *WorkerClient) StartWorkflow(ctx context.Context, clientName string,
	options client.StartWorkflowOptions, workflowFunc interface{}, args ...interface{}) (*workflow.Execution, error) {
	return c.clients[clientName].StartWorkflow(ctx, options, workflowFunc, args...)
}

// ExecuteWorkflow execute a workflow in a specific client
func (c *WorkerClient) ExecuteWorkflow(ctx context.Context, clientName string,
	options client.StartWorkflowOptions, workflowFunc interface{}, args ...interface{}) (client.WorkflowRun, error) {
	return c.clients[clientName].ExecuteWorkflow(ctx, options, workflowFunc, args...)
}
