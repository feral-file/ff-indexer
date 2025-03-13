package main

import "time"

type Stage int8

const (
	NftEventStageDone         = 0
	NftEventStageInit         = 1
	NftEventStageFullSync     = 2
	NftEventStageNotification = 3

	// deprecated
	NftEventStageFeed = 4

	NftEventStageTokenSaleIndexing = 5

	NftEventStageDoubleSync = 11
)

var NftEventStages = map[Stage]string{
	NftEventStageInit:              "stage_1_init",
	NftEventStageFullSync:          "stage_2_full_sync",
	NftEventStageNotification:      "stage_3_send_notification",
	NftEventStageFeed:              "stage_4_send_to_feed",
	NftEventStageTokenSaleIndexing: "stage_5_index_token_sale",
	NftEventStageDoubleSync:        "stage_11_double_sync_token",
}

const (
	SeriesRegistryEventStageDone = 0
	SeriesRegistryEventStageInit = 1
)

var SeriesEventStages = map[Stage]string{
	SeriesRegistryEventStageInit: "stage_1_init",
}

const DefaultCheckInterval = 10 * time.Second
const DefaultEventExpiryDays = 30
