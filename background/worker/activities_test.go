package worker

import (
	"reflect"
	"testing"

	"github.com/bitmark-inc/tzkt-go"
)

func TestDecodeParameterValueFirstPattern(t *testing.T) {
	mapArray := []map[string]interface{}{
		{
			"txs": []map[string]interface{}{
				{
					"to_":      "tz1NXE3jaTm4zksJPp6M3vQZGNqdAZG7b62b",
					"amount":   "1",
					"token_id": "2",
				},
			},
			"from_": "tz1gSFmDZcTrjNaYAMNQys1ufnqrKcWcPC2G",
		},
	}

	expected := []tzkt.ParametersValue{
		{
			From: "tz1gSFmDZcTrjNaYAMNQys1ufnqrKcWcPC2G",
			Txs: []tzkt.TxsFormat{
				{
					To:      "tz1NXE3jaTm4zksJPp6M3vQZGNqdAZG7b62b",
					Amount:  "1",
					TokenID: "2",
				},
			},
		},
	}

	result, err := decodeParametersValue(mapArray)
	if err != nil {
		t.Fatalf("Error decoding maps: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Decoded result does not match expected.\nGot: %+v\nExpected: %+v", result, expected)
	}
}

func TestDecodeParameterValueSecondPattern(t *testing.T) {
	mapArray := []map[string]interface{}{
		{
			"list": []map[string]interface{}{
				{
					"to":       "tz1SKUGqfkVPe6xM5xkUiJwuSsmzswD2Rk41",
					"amount":   "1",
					"token_id": "7373",
				},
			},
			"address": "tz1dXJ4JuwTVq2uidgq6hxVgqvderghd1F5i",
		},
	}

	expected := []tzkt.ParametersValue{
		{
			From: "tz1dXJ4JuwTVq2uidgq6hxVgqvderghd1F5i",
			Txs: []tzkt.TxsFormat{
				{
					To:      "tz1SKUGqfkVPe6xM5xkUiJwuSsmzswD2Rk41",
					Amount:  "1",
					TokenID: "7373",
				},
			},
		},
	}

	result, err := decodeParametersValue(mapArray)
	if err != nil {
		t.Fatalf("Error decoding maps: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Decoded result does not match expected.\nGot: %+v\nExpected: %+v", result, expected)
	}
}
