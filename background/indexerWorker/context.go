package indexerWorker

import (
	"time"

	"go.uber.org/cadence/workflow"
)

func ContextRetryActivity(ctx workflow.Context) workflow.Context {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Hour,
		RetryPolicy: &workflow.RetryPolicy{
			InitialInterval:    7,
			BackoffCoefficient: 1,
			MaximumAttempts:    5,
		},
	}

	return workflow.WithActivityOptions(ctx, ao)
}

func ContextNoRetryActivity(ctx workflow.Context) workflow.Context {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	return workflow.WithActivityOptions(ctx, ao)
}
