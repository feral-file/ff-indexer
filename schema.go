package indexer

import "time"

type AccountIdentity struct {
	AccountNumber   string    `json:"account_number" bson:"account_number"`
	Blockchain      string    `json:"blockchain" bson:"blockchain"`
	Name            string    `json:"name" bson:"name"`
	LastUpdatedTime time.Time `json:"-" bson:"lastUpdatedTime"`
}
