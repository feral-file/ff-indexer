package indexer

import "time"

type AccountIdentity struct {
	AccountNumber   string    `json:"accountNumber" bson:"accountNumber"`
	Blockchain      string    `json:"blockchain" bson:"blockchain"`
	Name            string    `json:"name" bson:"name"`
	LastUpdatedTime time.Time `json:"-" bson:"lastUpdatedTime"`
}
