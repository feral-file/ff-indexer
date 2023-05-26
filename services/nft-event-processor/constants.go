package main

import "time"

var EventStages = map[int8]string{
	1: "stage_1_update_the_latest_owner",
	2: "stage_2_full_sync",
	3: "stage_3_send_notification",
	4: "stage_4_send_to_feed",
}

const DefaultCheckInterval = 10 * time.Second
