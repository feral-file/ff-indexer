package tzkt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetContractToken(t *testing.T) {
	tc := New("")

	token, err := tc.GetContractToken("KT1LjmAdYQCLBjwv4S2oFkEzyHVkomAf5MrW", "24216")
	assert.NoError(t, err)
	assert.Equal(t, token.Contract.Alias, "Versum Items")

	token2, err := tc.GetContractToken("KT1NVvPsNDChrLRH5K2cy6Sc9r1uuUwdiZQd", "5084") // token with string formats
	assert.NoError(t, err)
	assert.Len(t, token2.Metadata.Formats, 3)

	token3, err := tc.GetContractToken("KT1RJ6PbjHpwc3M5rw5s2Nbmefwbuwbdxton", "777619")
	assert.NoError(t, err)
	assert.Len(t, token3.Metadata.Formats, 3)
}

func TestRetrieveTokens(t *testing.T) {
	tc := New("")

	ownedTokens, err := tc.RetrieveTokens("tz1RBi5DCVBYh1EGrcoJszkte1hDjrFfXm5C", time.Time{}, 0)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(ownedTokens), 1)
	assert.GreaterOrEqual(t, ownedTokens[0].Balance, int64(1))
}

func TestGetTokenTransfers(t *testing.T) {
	tc := New("")

	transfers, err := tc.GetTokenTransfers("KT1U6EHmNxJTkvaWJ4ThczG4FSDaHC21ssvi", "905625")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(transfers), 1)
	assert.Nil(t, transfers[0].From)
	assert.Equal(t, transfers[0].TransactionID, uint64(251825029644288))
	assert.Nil(t, transfers[0].From)
	assert.Equal(t, transfers[0].To.Address, "tz1QnNR17RHvXxDKHQEdRaAxrGL9hGysVcqT")
}

func TestGetTransaction(t *testing.T) {
	tc := New("")

	transaction, err := tc.GetTransaction(251825029644288)
	assert.NoError(t, err)
	assert.Equal(t, transaction.Hash, "ooJe9soP53x4dSBZR2mkEi1h3oQDCk5WZLaDBTVB3YzouC7dacQ")
}

func TestGetTokenActivityTime(t *testing.T) {
	tc := New("")

	activityTime, err := tc.GetTokenLastActivityTime("KT1U6EHmNxJTkvaWJ4ThczG4FSDaHC21ssvi", "905625")
	assert.NoError(t, err)

	activityTestTime := time.Unix(1655686019, 0)
	assert.GreaterOrEqual(t, activityTime.Sub(activityTestTime), time.Duration(0))
}

func TestGetTokenActivityTimeNotExist(t *testing.T) {
	tc := New("")

	activityTime, err := tc.GetTokenLastActivityTime("KT1U6EHmNxJTkvaWJ4ThczG4FSDaHC21ssvi", "0")
	assert.Error(t, err, "no activities for this token")
	assert.Equal(t, activityTime, time.Time{})
}

func TestGetTokenBalanceForOwner(t *testing.T) {
	tc := New("")

	owner, lastTime, err := tc.GetTokenBalanceAndLastTimeForOwner("KT1RJ6PbjHpwc3M5rw5s2Nbmefwbuwbdxton", "751194", "tz1bpvbjRGW1XHkALp4hFee6PKbnZCcoN9hE")
	assert.NoError(t, err)
	assert.Equal(t, owner, int64(1))
	assert.NotEqual(t, lastTime, time.Time{})
}

func TestGetArtworkMIMEType(t *testing.T) {
	tc := New("")

	token, err := tc.GetContractToken("KT1XXcp2U2vAn4dENmKjJkyYb8svTEf2DxTY", "0")
	assert.NoError(t, err)
	assert.Len(t, token.Metadata.Formats, 3)
	var mimeType string
	for _, f := range token.Metadata.Formats {
		if f.URI == token.Metadata.ArtifactURI {
			mimeType = string(f.MIMEType)
			break
		}
	}

	assert.Equal(t, mimeType, "image/jpeg")
}
func TestGetMIMETypeInArrayFormat(t *testing.T) {
	tc := New("")

	token, err := tc.GetContractToken("KT1Q4SBM941oAeu69v8LsrfwSiEkhMWJiVrp", "105353509316641797498497312618436889009736347208140239997663486800489418099672")
	assert.NoError(t, err)
	assert.Len(t, token.Metadata.Formats, 3)
	assert.Equal(t, "video/mp4", string(token.Metadata.Formats[0].MIMEType))
	assert.Equal(t, "image/jpeg", string(token.Metadata.Formats[1].MIMEType))
}

func TestHugeAmount(t *testing.T) {
	tc := New("")

	accountTokenTime, err := time.Parse(time.RFC3339, "2022-10-01T09:00:00Z")
	assert.NoError(t, err)

	_, err = tc.RetrieveTokens("tz1LiKcgzMA8E75vHtrr3wLk5Sx7r3GyMDNe", accountTokenTime, 0)
	assert.NoError(t, err)

	token, err := tc.GetContractToken("KT1F8gkt9o4a2DKwHVsZv9akrF7ZbaYBHpMy", "0")
	assert.NoError(t, err)
	assert.Equal(t, int64(token.TotalSupply), int64(-1))
}

func TestGetTokenOwners(t *testing.T) {
	tc := New("")

	var startTime time.Time
	var querLimit = 50

	owners, err := tc.GetTokenOwners("KT1RJ6PbjHpwc3M5rw5s2Nbmefwbuwbdxton", "784317", querLimit, startTime)
	assert.NoError(t, err)
	assert.Len(t, owners, querLimit)
	assert.Equal(t, owners[querLimit-1].Address, "tz1ZMDCUyEvsQykWuxTkBzVDcTzMtUEwTeiw")
	assert.Equal(t, owners[querLimit-1].LastTime.Format(time.RFC3339), "2022-10-01T21:24:59Z")
}

func TestGetTokenOwnersNow(t *testing.T) {
	tc := New("")

	var querLimit = 50

	owners, err := tc.GetTokenOwners("KT1RJ6PbjHpwc3M5rw5s2Nbmefwbuwbdxton", "784317", querLimit, time.Now().Add(-time.Hour))
	assert.NoError(t, err)
	assert.Len(t, owners, 0)
}
