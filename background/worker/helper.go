package worker

import (
	"context"
	"fmt"
	"strconv"
	"time"

	uberCadence "go.uber.org/cadence"
	"go.uber.org/cadence/.gen/go/shared"
	cadenceClient "go.uber.org/cadence/client"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	utils "github.com/bitmark-inc/autonomy-utils"
	"github.com/bitmark-inc/nft-indexer/cadence"
)

// StartIndexTokenWorkflow starts a workflow to index a single token
func StartIndexTokenWorkflow(c context.Context, client *cadence.WorkerClient, owner, contract, tokenID string, indexProvenance, indexPreview bool) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           fmt.Sprintf("index-single-nft-%s-%s", contract, tokenID),
		TaskList:                     TaskListName,
		ExecutionStartToCloseTimeout: 2 * time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyTerminateIfRunning,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext,
		w.IndexTokenWorkflow, owner, contract, tokenID, indexProvenance, indexPreview)
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

// StartIndexETHTokenWorkflow starts a workflow to index tokens for an ethereum address
func StartIndexETHTokenWorkflow(c context.Context, client *cadence.WorkerClient, caller string, owner string, includeHistory bool) {
	option := cadenceClient.StartWorkflowOptions{
		ID:                           WorkflowIDIndexTokenByOwner(caller, owner),
		TaskList:                     TaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, option, w.IndexETHTokenWorkflow, owner, includeHistory)
	if err != nil {
		log.Error("fail to start workflow to index token for owner", zap.Error(err), zap.String("caller", caller), zap.String("owner", owner))
	} else {
		log.Debug("start workflow for index ETH tokens from opensea for owner", zap.String("workflow_id", workflow.ID), zap.String("caller", caller), zap.String("owner", owner))
	}
}

// StartIndexTezosTokenWorkflow starts a workflow to index tokens for an ethereum address
func StartIndexTezosTokenWorkflow(c context.Context, client *cadence.WorkerClient, caller string, owner string, includeHistory bool) {
	option := cadenceClient.StartWorkflowOptions{
		ID:                           WorkflowIDIndexTokenByOwner(caller, owner),
		TaskList:                     TaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, option, w.IndexTezosTokenWorkflow, owner, includeHistory)
	if err != nil {
		log.Error("fail to start workflow to index token for owner", zap.Error(err), zap.String("caller", caller), zap.String("owner", owner))
	} else {
		log.Debug("start workflow for index Tezos tokens from opensea for owner", zap.String("workflow_id", workflow.ID), zap.String("caller", caller), zap.String("owner", owner))
	}
}

// StartRefreshTokenProvenanceByOwnerWorkflow starts a workflow to refresh token provenance by owner
func StartRefreshTokenProvenanceByOwnerWorkflow(c context.Context, client *cadence.WorkerClient, caller string, owner string) {
	option := cadenceClient.StartWorkflowOptions{
		ID:                           WorkflowIDRefreshTokenProvenanceByOwner(caller, owner),
		TaskList:                     ProvenanceTaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, option, w.RefreshTokenProvenanceByOwnerWorkflow, owner)
	if err != nil {
		log.Error("fail to start workflow to index token for owner", zap.Error(err), zap.String("caller", caller), zap.String("owner", owner))
	} else {
		log.Debug("start workflow for index Tezos tokens from opensea for owner", zap.String("workflow_id", workflow.ID), zap.String("caller", caller), zap.String("owner", owner))
	}
}

func StartRefreshTokenOwnershipWorkflow(c context.Context, client *cadence.WorkerClient,
	caller string, indexID string, delay time.Duration) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           WorkflowIDIndexTokenOwnershipByIndexID(caller, indexID),
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
		ID:                           WorkflowIDIndexTokenProvenanceByIndexID(caller, indexID),
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

func StartPendingTxFollowUpWorkflow(c context.Context, client *cadence.WorkerClient, delay time.Duration) error {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           "pending-tx-follow-up",
		TaskList:                     AccountTokenTaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext, w.PendingTxFollowUpWorkflow, delay)
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

// StartIndexTezosTokenWorkflow starts a workflow to index tokens for an ethereum address
func StartIndexTezosCollectionWorkflow(c context.Context, client *cadence.WorkerClient, caller string, creator string) {
	option := cadenceClient.StartWorkflowOptions{
		ID:                           WorkflowIDIndexCollectionsByOwner(caller, creator),
		TaskList:                     TaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, option, w.IndexTezosCollectionWorkflow, creator)
	if err != nil {
		log.Error("fail to start workflow to index ETH collections for owner", zap.Error(err), zap.String("caller", caller), zap.String("creator", creator))
	} else {
		log.Debug("start workflow for index ETH collections for owner", zap.String("workflow_id", workflow.ID), zap.String("caller", caller), zap.String("creator", creator))
	}
}

// StartIndexTezosTokenWorkflow starts a workflow to index tokens for an ethereum address
func StartIndexETHCollectionWorkflow(c context.Context, client *cadence.WorkerClient, caller string, creator string) {
	option := cadenceClient.StartWorkflowOptions{
		ID:                           WorkflowIDIndexCollectionsByOwner(caller, creator),
		TaskList:                     TaskListName,
		ExecutionStartToCloseTimeout: 3 * time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, option, w.IndexETHCollectionWorkflow, creator)
	if err != nil {
		log.Error("fail to start workflow to index ETH collections for owner", zap.Error(err), zap.String("caller", caller), zap.String("creator", creator))
	} else {
		log.Debug("start workflow for index ETH collections for owner", zap.String("workflow_id", workflow.ID), zap.String("caller", caller), zap.String("creator", creator))
	}
}

// StartIndexingTokenSale starts a workflow to index a token sale
func StartIndexingTokenSale(
	ctx context.Context,
	client *cadence.WorkerClient,
	blockchain string,
	txID string) error {
	opts := cadenceClient.StartWorkflowOptions{
		TaskList:                     TaskListName,
		ExecutionStartToCloseTimeout: 30 * time.Minute,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
		RetryPolicy: &uberCadence.RetryPolicy{
			InitialInterval:    15 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    5,
		},
	}

	var w NFTIndexerWorker
	switch blockchain {
	case utils.EthereumBlockchain:
		opts.ID = fmt.Sprintf("IndexEthereumTokenSale-%s", txID)
		exec, err := client.StartWorkflow(
			ctx,
			ClientName,
			opts,
			w.IndexEthereumTokenSale,
			txID,
			true)
		if nil != err {
			return err
		}
		log.Info("start workflow for indexing ethereum sale",
			zap.String("workflow_id", exec.ID),
			zap.String("run_id", exec.RunID))
	case utils.TezosBlockchain:
		opts.ID = fmt.Sprintf("IndexTezosTokenSaleFromTzktTxID-%s", txID)
		tzktTxID, err := strconv.ParseUint(txID, 10, 64)
		if nil != err {
			return err
		}

		exec, err := client.StartWorkflow(
			ctx,
			ClientName,
			opts,
			w.IndexTezosTokenSaleFromTzktTxID,
			tzktTxID)
		if nil != err {
			return err
		}
		log.Debug("start workflow for indexing tezos sale",
			zap.String("workflow_id", exec.ID),
			zap.String("run_id", exec.RunID))
	default:
		return fmt.Errorf("unsupported blockchain: %s", blockchain)
	}

	return nil
}
