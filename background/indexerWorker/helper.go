package indexerWorker

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	uberCadence "go.uber.org/cadence"
	"go.uber.org/cadence/.gen/go/shared"
	cadenceClient "go.uber.org/cadence/client"

	"github.com/bitmark-inc/nft-indexer/cadence"
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
		log.WithError(err).WithField("owner", owner).WithField("contract", contract).WithField("token_id", tokenID).
			Error("fail to start indexing workflow")
	} else {
		log.WithField("owner", owner).WithField("contract", contract).WithField("token_id", tokenID).WithField("workflow_id", workflow.ID).
			Debug("start workflow to index a token")
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
		log.WithError(err).WithField("caller", caller).Error("fail to start refreshing ownership workflow")
	} else {
		log.WithField("caller", caller).WithField("workflow_id", workflow.ID).Debug("start workflow for refreshing ownership")
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
		log.WithError(err).WithField("caller", caller).Error("fail to start refreshing provenance workflow")
	} else {
		log.WithField("caller", caller).WithField("workflow_id", workflow.ID).Debug("start workflow for refreshing provenance")
	}
}

func StartUpdateAccountTokensWorkflow(c context.Context, client *cadence.CadenceWorkerClient, delay time.Duration) error {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           "update-account-token-helper",
		TaskList:                     AccountTokenTaskListName,
		ExecutionStartToCloseTimeout: 20 * 365 * 24 * time.Hour, // fake infinite duration
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext, w.UpdateAccountTokensWorkflow, delay)
	if err != nil {
		log.WithError(err).Error("fail to start updating account token workflow")
		_, isAlreadyStartedError := err.(*shared.WorkflowExecutionAlreadyStartedError)
		if !isAlreadyStartedError {
			return err
		}
	} else {
		log.WithField("workflow_id", workflow.ID).Debug("start workflow for updating pending account tokens")
	}

	return nil
}

func StartUpdateSuggestedMIMETypeCronWorkflow(c context.Context, client *cadence.CadenceWorkerClient, delay time.Duration) error {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           "update-token-suggested-mime-type",
		TaskList:                     AccountTokenTaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		CronSchedule:                 "0 * * * *", //every hour
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext, w.UpdateSuggestedMIMETypeWorkflow, delay)
	if err != nil {
		log.WithError(err).Error("fail to start updating suggested mime type workflow")
		_, isAlreadyStartedError := err.(*shared.WorkflowExecutionAlreadyStartedError)
		if !isAlreadyStartedError {
			return err
		}
	} else {
		log.WithField("workflow_id", workflow.ID).Debug("start workflow for updating suggested mime type")
	}

	return nil
}
