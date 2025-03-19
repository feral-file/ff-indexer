package main

import (
	"context"

	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
)

// logStageEvent logs events by stages
func (e *EventProcessor) logStageEvent(ctx context.Context, stage Stage, message string, fields ...zap.Field) {
	fields = append(fields, zap.Int8("stage", int8(stage)))
	log.InfoWithContext(ctx, message, fields...)
}

// logStartStage log when start a stage
func (e *EventProcessor) logStartStage(ctx context.Context, eventID string, stage Stage) {
	log.InfoWithContext(ctx, "start stage for event: ", zap.Int8("stage", int8(stage)), zap.Any("event", eventID))
}

// logEndint8(stage) log when end a stage
func (e *EventProcessor) logEndStage(ctx context.Context, eventID string, stage Stage) {
	log.InfoWithContext(ctx, "finished stage for event: ", zap.Int8("stage", int8(stage)), zap.Any("event", eventID))
}
