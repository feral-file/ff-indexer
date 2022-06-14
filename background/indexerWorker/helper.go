package indexerWorker

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	cadenceClient "go.uber.org/cadence/client"

	"github.com/bitmark-inc/nft-indexer/cadence"
)

func StartIndexTokenWorkflow(c context.Context, client *cadence.CadenceWorkerClient, owner, contract, tokenID string) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           fmt.Sprintf("index-single-nft-%s-%s", contract, tokenID),
		TaskList:                     TaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w NFTIndexerWorker

	workflow, err := client.StartWorkflow(c, ClientName, workflowContext,
		w.IndexTokenWorkflow, owner, contract, tokenID)
	if err != nil {
		log.WithError(err).WithField("owner", owner).WithField("contract", contract).WithField("token_id", tokenID).
			Error("fail to start indexing workflow")
	} else {
		log.WithField("owner", owner).WithField("contract", contract).WithField("token_id", tokenID).WithField("workflow_id", workflow.ID).
			Info("start workflow for indexing a token")
	}
}
