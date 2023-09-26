package main

import (
	"time"

	"github.com/bitmark-inc/tzkt-go"
)

type EventType string

const (
	EventTypeMint         EventType = "mint"
	EventTypeBurned       EventType = "burned"
	EventTypeTransfer     EventType = "transfer"
	EventTypeTokenUpdated EventType = "token_updated"
)

type TokenTransferResponse struct {
	Type  int                  `json:"type"`
	Data  []tzkt.TokenTransfer `json:"data"`
	State int64                `json:"state"`
}

type BigmapUpdateResponse struct {
	Type  int                 `json:"type"`
	Data  []tzkt.BigmapUpdate `json:"data"`
	State int64               `json:"state"`
}

type TokenEvent struct {
	EventType       EventType
	From            string
	To              string
	ContractAddress string
	Blockchain      string
	TokenID         string
	TxID            string
	TxTime          time.Time
	Level           uint64
}
