package indexerWorker

import (
	"context"
	"fmt"
	"time"

	uberCadence "go.uber.org/cadence"
	cadenceClient "go.uber.org/cadence/client"
	"go.uber.org/zap"

	"github.com/bitmark-inc/nft-indexer/cadence"
	log "github.com/bitmark-inc/nft-indexer/zapLog"
)

func StartIndexTokenWorkflow(c context.Context, client *cadence.CadenceWorkerClient, owner, contract, tokenID string, indexPreview bool) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           fmt.Sprintf("index-single-nft-%s-%s", contract, tokenID),
		TaskList:                     TaskListName,
		ExecutionStartToCloseTimeout: 2 * time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext,
		w.IndexTokenWorkflow, owner, contract, tokenID, indexPreview)
	if err != nil {
		log.Logger.Error("fail to start indexing workflow",
			zap.Error(err),
			zap.String("owner", owner), zap.String("contract", contract), zap.String("token_id", tokenID))

	} else {
		log.Logger.Debug("start workflow to index a token",
			zap.String("owner", owner),
			zap.String("contract", contract),
			zap.String("token_id", tokenID),
			zap.String("workflow_id", workflow.ID))

	}
}

func StartRefreshTokenOwnershipWorkflow(c context.Context, client *cadence.CadenceWorkerClient,
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
		log.Logger.Error("fail to start refreshing ownership workflow", zap.Error(err), zap.String("caller", caller))
	} else {
		log.Logger.Debug("start workflow for refreshing ownership", zap.String("caller", caller), zap.String("workflow_id", workflow.ID))
	}
}

func StartRefreshTokenProvenanceWorkflow(c context.Context, client *cadence.CadenceWorkerClient,
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
		log.Logger.Error("fail to start refreshing provenance workflow", zap.Error(err), zap.String("caller", caller))
	} else {
		log.Logger.Debug("start workflow for refreshing provenance", zap.String("caller", caller), zap.String("workflow_id", workflow.ID))
	}
}

func StartUpdateAccountTokensWorkflow(c context.Context, client *cadence.CadenceWorkerClient, delay time.Duration) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           fmt.Sprintf("update-account-token-helper-%s", time.Now()),
		TaskList:                     AccountTokenTaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext, w.UpdateAccountTokensWorkflow, delay)
	if err != nil {
		log.Logger.Error("fail to start updating account token workflow", zap.Error(err))
	} else {
		log.Logger.Debug("start workflow for updating pending account tokens", zap.String("workflow_id", workflow.ID))
	}

}
