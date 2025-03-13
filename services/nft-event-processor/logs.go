package main

import (
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
)

// logStageEvent logs events by stages
func (e *EventProcessor) logStageEvent(stage Stage, message string, fields ...zap.Field) {
	fields = append(fields, zap.Int8("stage", int8(stage)))
	log.Info(message, fields...)
}

// logStartStage log when start a stage
func (e *EventProcessor) logStartStage(eventID string, stage Stage) {
	log.Info("start stage for event: ", zap.Int8("stage", int8(stage)), zap.Any("event", eventID))
}

// logEndint8(stage) log when end a stage
func (e *EventProcessor) logEndStage(eventID string, stage Stage) {
	log.Info("finished stage for event: ", zap.Int8("stage", int8(stage)), zap.Any("event", eventID))
}
