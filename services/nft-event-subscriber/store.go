package main

import (
	"context"
	"time"

	indexer "github.com/bitmark-inc/nft-indexer"
)

func (s *NFTEventSubscriber) GetAccountIDByAddress(address string) ([]string, error) {
	return s.accountStore.GetAccountIDByAddress(address)
}

func (s *NFTEventSubscriber) GetTokensByIndexID(c context.Context, indexID string) (*indexer.Token, error) {
	tokens, err := s.store.GetTokensByIndexIDs(c, []string{indexID})
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return nil, nil
	}

	return &tokens[0], err
}

func (s *NFTEventSubscriber) UpdateOwner(c context.Context, id, owner string, updatedAt time.Time) error {
	return s.store.UpdateOwner(c, id, owner, updatedAt)
}

func (s *NFTEventSubscriber) UpdateAccountTokenOwners(c context.Context, indexID string, lastActivityTime time.Time, ownerBalances []indexer.OwnerBalances) error {
	return s.store.UpdateAccountTokenOwners(c, indexID, lastActivityTime, ownerBalances)
}
