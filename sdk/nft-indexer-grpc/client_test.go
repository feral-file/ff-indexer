package sdk

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetTokensByIndexID(t *testing.T) {
	client, err := NewIndexerClient("localhost:8889")
	if err != nil {
		fmt.Println(err)
		return
	}

	token, err := client.GetTokensByIndexID(
		context.Background(),
		"tez-KT1VZ6Zkoae9DtXkbuw4wtFCg9WH8eywcvEX-23798030035473632618901897089878275372960165372586891230635421889000008911882"
		)

	fmt.Println("token: ", token)
	assert.NoError(t, err)
}
