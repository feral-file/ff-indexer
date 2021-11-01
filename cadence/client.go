package cadence

import (
	"context"

	"github.com/spf13/viper"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/workflow"
)

var CadenceService = "cadence-frontend"

// CadenceWorkerClient manages multiple cadence worker service clients
type CadenceWorkerClient struct {
	Domain string

	clients map[string]client.Client
}

func NewWorkerClient(domain string) *CadenceWorkerClient {
	return &CadenceWorkerClient{
		Domain:  domain,
		clients: map[string]client.Client{},
	}
}

// AddService register a service client
func (c *CadenceWorkerClient) AddService(clientName string) {
	serviceClient := BuildCadenceServiceClient(viper.GetString("cadence.host_port"), clientName, CadenceService)

	cadenceWorker := client.NewClient(
		serviceClient,
		c.Domain,
		&client.Options{},
	)

	c.clients[clientName] = cadenceWorker
}

// StartWorkflow triggers a workflow in a specific client
func (c *CadenceWorkerClient) StartWorkflow(ctx context.Context, clientName string,
	options client.StartWorkflowOptions, workflowFunc interface{}, args ...interface{}) (*workflow.Execution, error) {
	return c.clients[clientName].StartWorkflow(ctx, options, workflowFunc, args...)
}
