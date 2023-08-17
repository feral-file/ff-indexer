package ens

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveDomain(t *testing.T) {
	infuraProjectID := os.Getenv("INFURA_PROJECT_ID")
	ens := New("https://mainnet.infura.io/v3/" + infuraProjectID)
	name, err := ens.ResolveDomain("0x99fc8AD516FBCC9bA3123D56e63A35d05AA9EFB8")
	assert.NoError(t, err)
	assert.Equal(t, "einstein-rosen.eth", name)
}
