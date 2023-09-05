package main

import "time"

type TokenTransferResponse struct {
	Type  int             `json:"type"`
	Data  []TokenTransfer `json:"data"`
	State int64           `json:"state"`
}

type TokenTransfer struct {
	Timestamp     time.Time `json:"timestamp"`
	Level         uint64    `json:"level"`
	TransactionID uint64    `json:"transactionId"`
	Amount        string    `json:"amount"`
	Token         Token     `json:"token"`
	From          *Account  `json:"from"`
	To            Account   `json:"to"`
}

type Account struct {
	Address string `json:"address"`
}

type Token struct {
	ID       int64   `json:"id"`
	Contract Account `json:"contract"`
	TokenID  string  `json:"tokenId"`
}
