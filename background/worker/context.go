package worker

import (
	"time"

	"go.uber.org/cadence"
	cadenceClient "go.uber.org/cadence/client"
	"go.uber.org/cadence/workflow"
)

// ContextRetryActivity returns an activity context for retrying task
func ContextRetryActivity(ctx workflow.Context, taskList string) workflow.Context {
	ao := workflow.ActivityOptions{
		TaskList:               taskList,
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Hour,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    10 * time.Second,
			BackoffCoefficient: 1,
			MaximumAttempts:    6,
		},
	}

	return workflow.WithActivityOptions(ctx, ao)
}

func ContextFastActivity(ctx workflow.Context, taskList string) workflow.Context {
	ao := workflow.ActivityOptions{
		TaskList:               taskList,
		ScheduleToStartTimeout: 10 * time.Second,
		StartToCloseTimeout:    1 * time.Minute,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 1.0,
			MaximumAttempts:    6,
		},
	}

	if taskList != "" {
		ao.TaskList = taskList
	}

	return workflow.WithActivityOptions(ctx, ao)
}

// ContextRegularActivity returns a regular activity context
func ContextRegularActivity(ctx workflow.Context, taskList string) workflow.Context {
	ao := workflow.ActivityOptions{
		TaskList:               taskList,
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 1.0,
			MaximumAttempts:    10,
		},
	}

	if taskList != "" {
		ao.TaskList = taskList
	}

	return workflow.WithActivityOptions(ctx, ao)
}

// ContextRegularChildWorkflow returns a regular child workflow context
func ContextRegularChildWorkflow(ctx workflow.Context, taskList string) workflow.Context {
	return workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		TaskList:                     taskList,
		ExecutionStartToCloseTimeout: 10 * time.Minute,
	})
}

// ContextNamedRegularChildWorkflow returns a named regular child workflow context
func ContextNamedRegularChildWorkflow(ctx workflow.Context, workflowID, taskList string) workflow.Context {
	return workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:                   workflowID,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
		TaskList:                     taskList,
		ExecutionStartToCloseTimeout: 10 * time.Minute,
	})
}

// ContextSlowChildWorkflow returns a regular child workflow context
func ContextSlowChildWorkflow(ctx workflow.Context, taskList string) workflow.Context {
	return workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		TaskList:                     taskList,
		ExecutionStartToCloseTimeout: time.Hour,
	})
}

// ContextDetachedChildWorkflow returns a child workflow context that allows to detach from its parent
func ContextDetachedChildWorkflow(ctx workflow.Context, workflowID, taskList string) workflow.Context {
	return workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:                   workflowID,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
		TaskList:                     taskList,
		ExecutionStartToCloseTimeout: time.Hour,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:    10 * time.Second,
			BackoffCoefficient: 1.0,
			MaximumAttempts:    60,
		},
		ParentClosePolicy: cadenceClient.ParentClosePolicyAbandon,
	})
}
