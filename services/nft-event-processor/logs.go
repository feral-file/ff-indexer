package main

import (
	"go.uber.org/zap"

	"github.com/bitmark-inc/nft-indexer/log"
)

// logStageEvent logs events by stages
func (e *EventProcessor) logStageEvent(stage int8, message string, fields ...zap.Field) {
	fields = append(fields, zap.Int8("stage", stage))
	log.Info(message, fields...)
}

// logStartStage log when start a stage
func (e *EventProcessor) logStartStage(event NFTEvent, stage int8) {
	log.Info("start stage for event: ", zap.Int8("stage", stage), zap.Any("event", event.ID))
}

// logEndStage log when end a stage
func (e *EventProcessor) logEndStage(event NFTEvent, stage int8) {
	log.Info("finished stage for event: ", zap.Int8("stage", stage), zap.Any("event", event.ID))
}
