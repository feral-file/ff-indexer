package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	cadenceClient "go.uber.org/cadence/client"

	"github.com/bitmark-inc/nft-indexer/background/indexerWorker"
)

func (s *NFTEventSubscriber) startRefreshProvenanceWorkflow(c context.Context, refreshProvenanceTaskID string, indexIDs []string, delay time.Duration) {
	workflowContext := cadenceClient.StartWorkflowOptions{
		ID:                           fmt.Sprintf("index-token-%s-provenance", refreshProvenanceTaskID),
		TaskList:                     indexerWorker.TaskListName,
		ExecutionStartToCloseTimeout: time.Hour,
		WorkflowIDReusePolicy:        cadenceClient.WorkflowIDReusePolicyAllowDuplicate,
	}

	var w indexerWorker.NFTIndexerWorker

	workflow, err := s.cadenceWorker.StartWorkflow(c, indexerWorker.ClientName, workflowContext, w.RefreshTokenProvenanceWorkflow, indexIDs, delay)
	if err != nil {
		logrus.WithError(err).WithField("refreshProvenanceTaskID", refreshProvenanceTaskID).Error("fail to start refreshing provenance workflow")
	} else {
		logrus.WithField("refreshProvenanceTaskID", refreshProvenanceTaskID).WithField("workflow_id", workflow.ID).Info("start workflow for refreshing provenance")
	}
}
