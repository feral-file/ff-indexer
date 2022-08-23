package indexerWorker

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	cadenceClient "go.uber.org/cadence/client"

	"github.com/bitmark-inc/nft-indexer/cadence"
)

func StartIndexTokenWorkflow(c context.Context, client *cadence.CadenceWorkerClient, owner, contract, tokenID string, indexPreview bool) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           fmt.Sprintf("index-single-nft-%s-%s", contract, tokenID),
		TaskList:                     TaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext,
		w.IndexTokenWorkflow, owner, contract, tokenID, indexPreview)
	if err != nil {
		log.WithError(err).WithField("owner", owner).WithField("contract", contract).WithField("token_id", tokenID).
			Error("fail to start indexing workflow")
	} else {
		log.WithField("owner", owner).WithField("contract", contract).WithField("token_id", tokenID).WithField("workflow_id", workflow.ID).
			Debug("start workflow to index a token")
	}
}

func StartRefreshTokenOwnershipWorkflow(c context.Context, client *cadence.CadenceWorkerClient,
	refreshOwnershipTaskID string, indexID string, delay time.Duration) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           fmt.Sprintf("index-token-ownership-helper-%s", refreshOwnershipTaskID),
		TaskList:                     TaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext, w.RefreshTokenOwnershipWorkflow, []string{indexID}, delay)
	if err != nil {
		log.WithError(err).WithField("refreshOwnershipTaskID", refreshOwnershipTaskID).Error("fail to start refreshing ownership workflow")
	} else {
		log.WithField("refreshOwnershipTaskID", refreshOwnershipTaskID).WithField("workflow_id", workflow.ID).Debug("start workflow for refreshing ownership")
	}
}

func StartRefreshTokenProvenanceWorkflow(c context.Context, client *cadence.CadenceWorkerClient,
	refreshProvenanceTaskID string, indexID string, delay time.Duration) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           fmt.Sprintf("index-token-provenance-helper-%s", refreshProvenanceTaskID),
		TaskList:                     TaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext, w.RefreshTokenProvenanceWorkflow, []string{indexID}, delay)
	if err != nil {
		log.WithError(err).WithField("refreshProvenanceTaskID", refreshProvenanceTaskID).Error("fail to start refreshing provenance workflow")
	} else {
		log.WithField("refreshProvenanceTaskID", refreshProvenanceTaskID).WithField("workflow_id", workflow.ID).Debug("start workflow for refreshing provenance")
	}
}
