package main

import "time"

type Stage int8

const (
	StageDone         = 0
	StageInit         = 1
	StageFullSync     = 2
	StageNotification = 3

	// deprecated
	StageFeed = 4

	StageTokenSaleIndexing = 5

	StageDoubleSync = 11
)

var EventStages = map[Stage]string{
	StageInit:              "stage_1_init",
	StageFullSync:          "stage_2_full_sync",
	StageNotification:      "stage_3_send_notification",
	StageFeed:              "stage_4_send_to_feed",
	StageTokenSaleIndexing: "stage_5_index_token_sale",
	StageDoubleSync:        "stage_11_double_sync_token",
}

const DefaultCheckInterval = 10 * time.Second
const DefaultEventExpiryDays = 30
