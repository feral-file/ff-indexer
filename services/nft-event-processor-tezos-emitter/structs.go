package main

import "github.com/bitmark-inc/tzkt-go"

type TokenTransferResponse struct {
	Type  int                  `json:"type"`
	Data  []tzkt.TokenTransfer `json:"data"`
	State int64                `json:"state"`
}
