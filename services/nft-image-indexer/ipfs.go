package main

import (
	"time"

	"github.com/bitmark-inc/nft-indexer/externals/pinata"
	"github.com/spf13/viper"
)

// PinnedFile defines functions of IPFSPinService
type IPFSPinService interface {
	Pin(cid string) (bool, error)
}

// PinataIPFSPinService is an IPFSPinService implementation using pinata
type PinataIPFSPinService struct {
	client *pinata.PinataAPIClient
}

func NewPinataIPFSPinService() *PinataIPFSPinService {
	return &PinataIPFSPinService{
		client: pinata.New("api.pinata.cloud", viper.GetString("pinata.jwt"), 10*time.Second),
	}
}

// Pin a file to IPFS through a CID
func (p *PinataIPFSPinService) Pin(cid string) (bool, error) {
	pinnedFile, err := p.client.PinnedFile(cid)
	if err != nil {
		return false, err
	}

	// file has already pinned
	if pinnedFile != nil {
		return true, nil
	}

	job, err := p.client.PinJobs(cid)
	if err != nil {
		return false, err
	}

	// the file is pinning
	if job != nil {
		return false, nil
	}

	_, err = p.client.PinByHash(cid, nil)
	return false, err
}
