package indexer

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalValidBlockchainAddress(t *testing.T) {
	testFixtures := map[string]string{
		"0x70460be6b2ad5b900371601a2867eadbfef572ce":         "0x70460bE6b2ad5B900371601a2867EAdBFeF572cE",
		"0x6023e55814dc00f094386d4eb7e17ce49ab1a190":         "0x6023E55814DC00F094386d4eb7e17Ce49ab1A190",
		"tz1Td5qwQxz5mDZiwk7TsRGhDU2HBvXgULip":               "tz1Td5qwQxz5mDZiwk7TsRGhDU2HBvXgULip",
		"KT1U6EHmNxJTkvaWJ4ThczG4FSDaHC21ssvi":               "KT1U6EHmNxJTkvaWJ4ThczG4FSDaHC21ssvi",
		"a3ezwdYVEVrHwszQrYzDTCAZwUD3yKtNsCq9YhEu97bPaGAKy1": "a3ezwdYVEVrHwszQrYzDTCAZwUD3yKtNsCq9YhEu97bPaGAKy1",
	}

	for testAddress, expectAddress := range testFixtures {
		var addr BlockchainAddress

		assert.NoError(t, json.Unmarshal([]byte(fmt.Sprintf(`"%s"`, testAddress)), &addr))
		assert.Equal(t, expectAddress, string(addr))
	}
}
