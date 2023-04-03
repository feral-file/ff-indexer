package indexerWorker

import (
	"context"
	"fmt"
	"time"

	uberCadence "go.uber.org/cadence"
	"go.uber.org/cadence/.gen/go/shared"
	cadenceClient "go.uber.org/cadence/client"
	"go.uber.org/zap"

	"github.com/bitmark-inc/nft-indexer/cadence"
	"github.com/bitmark-inc/nft-indexer/log"
)

func StartIndexTokenWorkflow(c context.Context, client *cadence.WorkerClient, owner, contract, tokenID string, indexPreview bool, fromEvent string) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           fmt.Sprintf("index-single-nft-%s-%s", contract, tokenID),
		TaskList:                     TaskListName,
		ExecutionStartToCloseTimeout: 2 * time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext,
		w.IndexTokenWorkflow, owner, contract, tokenID, indexPreview, fromEvent)
	if err != nil {
		log.Error("fail to start indexing workflow",
			zap.Error(err),
			zap.String("owner", owner), zap.String("contract", contract), zap.String("token_id", tokenID))

	} else {
		log.Debug("start workflow to index a token",
			zap.String("owner", owner),
			zap.String("contract", contract),
			zap.String("token_id", tokenID),
			zap.String("workflow_id", workflow.ID))

	}
}

func StartRefreshTokenOwnershipWorkflow(c context.Context, client *cadence.WorkerClient,
	caller string, indexID string, delay time.Duration) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           WorkflowIDIndexTokenOwnershipByHelper(caller, indexID),
		TaskList:                     ProvenanceTaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext, w.RefreshTokenOwnershipWorkflow, []string{indexID}, delay)
	if err != nil {
		log.Error("fail to start refreshing ownership workflow", zap.Error(err), zap.String("caller", caller))
	} else {
		log.Debug("start workflow for refreshing ownership", zap.String("caller", caller), zap.String("workflow_id", workflow.ID))
	}
}

func StartRefreshTokenProvenanceWorkflow(c context.Context, client *cadence.WorkerClient,
	caller string, indexID string, delay time.Duration) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           WorkflowIDIndexTokenProvenanceByHelper(caller, indexID),
		TaskList:                     ProvenanceTaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
		RetryPolicy: &uberCadence.RetryPolicy{
			InitialInterval:    10 * time.Second,
			BackoffCoefficient: 1.0,
			MaximumAttempts:    60,
		},
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext, w.RefreshTokenProvenanceWorkflow, []string{indexID}, delay)
	if err != nil {
		log.Error("fail to start refreshing provenance workflow", zap.Error(err), zap.String("caller", caller))
	} else {
		log.Debug("start workflow for refreshing provenance", zap.String("caller", caller), zap.String("workflow_id", workflow.ID))
	}
}

func StartUpdateAccountTokensWorkflow(c context.Context, client *cadence.WorkerClient, delay time.Duration) error {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           "update-account-token-helper",
		TaskList:                     AccountTokenTaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext, w.UpdateAccountTokensWorkflow, delay)
	if err != nil {
		log.Error("fail to start updating account token workflow", zap.Error(err))
		_, isAlreadyStartedError := err.(*shared.WorkflowExecutionAlreadyStartedError)
		if !isAlreadyStartedError {
			return err
		}
	} else {
		log.Debug("start workflow for updating pending account tokens", zap.String("workflow_id", workflow.ID))
	}

	return nil
}

func StartUpdateSuggestedMIMETypeCronWorkflow(c context.Context, client *cadence.WorkerClient, delay time.Duration) error {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           "update-token-suggested-mime-type",
		TaskList:                     AccountTokenTaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		CronSchedule:                 "0 * * * *", //every hour
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext, w.UpdateSuggestedMIMETypeWorkflow, delay)
	if err != nil {
		log.Error("fail to start updating suggested mime type workflow", zap.Error(err))
		_, isAlreadyStartedError := err.(*shared.WorkflowExecutionAlreadyStartedError)
		if !isAlreadyStartedError {
			return err
		}
	} else {
		log.Debug("start workflow for updating suggested mime type", zap.String("workflow_id", workflow.ID))
	}

	return nil
}

func StartDetectAssetChangeWorkflow(c context.Context, client *cadence.WorkerClient) error {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           "detect-asset-change-helper",
		TaskList:                     AccountTokenTaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		CronSchedule:                 "0 * * * *", //every hour
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext, w.DetectAssetChangeWorkflow)
	if err != nil {
		log.Error("fail to start detect asset change workflow", zap.Error(err))
		_, isAlreadyStartedError := err.(*shared.WorkflowExecutionAlreadyStartedError)
		if !isAlreadyStartedError {
			return err
		}
	} else {
		log.Debug("start workflow for detecting asset change", zap.String("workflow_id", workflow.ID))
	}

	return nil
}
