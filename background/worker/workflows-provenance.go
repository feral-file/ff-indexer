package worker

import (
	"errors"
	"time"

	log "github.com/bitmark-inc/autonomy-logger"
	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

// refreshTokenProvenanceByOwnerDetachedWorkflow creates a detached workflow to trigger token provenance check
func (w *Worker) refreshTokenProvenanceByOwnerDetachedWorkflow(ctx workflow.Context, caller, owner string) {
	logger := log.CadenceWorkflowLogger(ctx)

	cwo := ContextDetachedChildWorkflow(ctx, WorkflowIDRefreshTokenProvenanceByOwner(caller, owner), ProvenanceTaskListName)

	var cw workflow.Execution
	if err := workflow.
		ExecuteChildWorkflow(cwo, w.RefreshTokenProvenanceByOwnerWorkflow, owner).
		GetChildWorkflowExecution().Get(ctx, &cw); err != nil {
		logger.Warn("fail to refresh token provenance by owner", zap.Error(err))
	}
}

// RefreshTokenProvenanceByOwnerWorkflow is a workflow to refresh provenance for a specific owner
func (w *Worker) RefreshTokenProvenanceByOwnerWorkflow(ctx workflow.Context, owner string) error {
	logger := log.CadenceWorkflowLogger(ctx)

	var ownedTokenIDs []string
	if err := workflow.ExecuteActivity(
		ContextRegularActivity(ctx, w.ProvenanceTaskListName),
		w.GetOwnedTokenIDsByOwner, owner,
	).Get(ctx, &ownedTokenIDs); err != nil {
		logger.Warn("fail to get owned token IDs by owner", zap.Error(err))
	}

	if err := workflow.ExecuteActivity(
		ContextRegularActivity(ctx, w.ProvenanceTaskListName),
		w.FilterTokenIDsWithInconsistentProvenanceForOwner, ownedTokenIDs, owner,
	).Get(ctx, &ownedTokenIDs); err != nil {
		logger.Warn("fail to filter token IDs with inconsistent provenance for owner", zap.Error(err))
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
			logger.Error(errors.New("fail to refresh token provenance"), zap.Error(err))
			return err
		}
	}

	return nil
}

// RefreshTokenProvenanceWorkflow is a workflow to refresh provenance for a specific token
func (w *Worker) RefreshTokenProvenanceWorkflow(ctx workflow.Context, indexIDs []string, delay time.Duration) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.ProvenanceTaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	return workflow.ExecuteActivity(ctx, w.RefreshTokenProvenance, indexIDs, delay).Get(ctx, nil)
}

// RefreshTokenOwnershipWorkflow is a workflow to refresh ownership for a specific token
func (w *Worker) RefreshTokenOwnershipWorkflow(ctx workflow.Context, indexIDs []string, delay time.Duration) error {
	ao := workflow.ActivityOptions{
		TaskList:               w.ProvenanceTaskListName,
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    time.Hour,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	return workflow.ExecuteActivity(ctx, w.RefreshTokenOwnership, indexIDs, delay).Get(ctx, nil)
}
