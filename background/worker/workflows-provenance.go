package worker

import (
	"time"

	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

// refreshTokenProvenanceByOwnerDetachedWorkflow creates a detached workflow to trigger token provenance check
func (w *NFTIndexerWorker) refreshTokenProvenanceByOwnerDetachedWorkflow(ctx workflow.Context, caller, owner string) {
	log := workflow.GetLogger(ctx)

	cwo := ContextDetachedChildWorkflow(ctx, WorkflowIDRefreshTokenProvenanceByOwner(caller, owner), ProvenanceTaskListName)

	var cw workflow.Execution
	if err := workflow.
		ExecuteChildWorkflow(cwo, w.RefreshTokenProvenanceByOwnerWorkflow, owner).
		GetChildWorkflowExecution().Get(ctx, &cw); err != nil {
		log.Error("fail to start workflow RefreshTokenProvenanceByOwnerWorkflow", zap.Error(err), zap.String("owner", owner))
	}
	log.Info("workflow RefreshTokenProvenanceByOwnerWorkflow started", zap.String("workflow_id", cw.ID), zap.String("owner", owner))
}

// RefreshTokenProvenanceByOwnerWorkflow is a workflow to refresh provenance for a specific owner
func (w *NFTIndexerWorker) RefreshTokenProvenanceByOwnerWorkflow(ctx workflow.Context, owner string) error {
	log := workflow.GetLogger(ctx)

	var ownedTokenIDs []string

	if err := workflow.ExecuteActivity(
		ContextRegularActivity(ctx, w.ProvenanceTaskListName),
		w.GetOwnedTokenIDsByOwner, owner,
	).Get(ctx, &ownedTokenIDs); err != nil {
		log.Error("fail to refresh provenance for indexIDs", zap.Error(err), zap.String("owner", owner))
	}

	if err := workflow.ExecuteActivity(
		ContextRegularActivity(ctx, w.ProvenanceTaskListName),
		w.FilterTokenIDsWithInconsistentProvenanceForOwner, ownedTokenIDs, owner,
	).Get(ctx, &ownedTokenIDs); err != nil {
		log.Error("fail to refresh provenance for indexIDs", zap.Error(err), zap.String("owner", owner))
	}

	batchIndexingTokens := 25

	for i := 0; i < len(ownedTokenIDs); i += batchIndexingTokens {
		endIndex := i + batchIndexingTokens
		if endIndex > len(ownedTokenIDs) {
			endIndex = len(ownedTokenIDs)
		}

		if err := workflow.ExecuteChildWorkflow(
			ContextSlowChildWorkflow(ctx, ProvenanceTaskListName),
			w.RefreshTokenProvenanceWorkflow, ownedTokenIDs[i:endIndex], 0,
		).Get(ctx, nil); err != nil {
			log.Error("fail to refresh provenance for indexIDs", zap.Error(err), zap.String("owner", owner))
			return err
		}
	}

	return nil
}

// RefreshTokenProvenanceWorkflow is a workflow to refresh provenance for a specific token
func (w *NFTIndexerWorker) RefreshTokenProvenanceWorkflow(ctx workflow.Context, indexIDs []string, delay time.Duration) error {
	ao := workflow.ActivityOptions{

		TaskList:               w.ProvenanceTaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	log := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Debug("start RefreshTokenProvenanceWorkflow")

	err := workflow.ExecuteActivity(ctx, w.RefreshTokenProvenance, indexIDs, delay).Get(ctx, nil)
	if err != nil {
		log.Error("fail to refresh provenance for indexIDs", zap.Error(err), zap.Any("indexIDs", indexIDs))
	}

	return err
}

// RefreshTokenOwnershipWorkflow is a workflow to refresh ownership for a specific token
func (w *NFTIndexerWorker) RefreshTokenOwnershipWorkflow(ctx workflow.Context, indexIDs []string, delay time.Duration) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.ProvenanceTaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}

	log := workflow.GetLogger(ctx)

	ctx = workflow.WithActivityOptions(ctx, ao)

	log.Debug("start RefreshTokenOwnershipWorkflow")

	err := workflow.ExecuteActivity(ctx, w.RefreshTokenOwnership, indexIDs, delay).Get(ctx, nil)
	if err != nil {
		log.Error("fail to refresh ownership for indexIDs", zap.Error(err), zap.Any("indexIDs", indexIDs))
	}

	return err
}
