package main

import (
	"fmt"

	"github.com/gin-gonic/gin"

	notification "github.com/bitmark-inc/autonomy-notification"
)

// notifyNewNFT notifies the arrival of a new token
func (s *NFTEventSubscriber) notifyNewNFT(accountID, toAddress, tokenID string) error {
	fmt.Println("\n\n ==> notifyNew tokenID: ", tokenID)
	return s.notification.SendNotification("",
		notification.NEW_NFT_ARRIVED,
		accountID,
		gin.H{
			"notification_type": "gallery_new_nft",
			"owner":             toAddress,
			"token_id":          tokenID,
		})
}
