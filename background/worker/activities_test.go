package worker

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/bitmark-inc/tzkt-go"
)

func TestDecodeParameterValueFirstPattern(t *testing.T) {
	data := `[
		{
			"txs":
			[
				{
					"to_": "tz1Wi5BHFA4qqr6cSXoQkpSKU8F1aMdL5cvs",
					"amount": "1",
					"token_id": "37214540304218121786566893708923600581837527203284427749671447415338838815459"
				}
			],
			"from_": "tz1e1yrqu7E42rMqxVt44mgnfsR6rhuJQ38i"
		}
	]`
	var mapArray interface{}
	json.Unmarshal([]byte(data), &mapArray)

	expected := []tzkt.ParametersValue{
		{
			From: "tz1e1yrqu7E42rMqxVt44mgnfsR6rhuJQ38i",
			Txs: []tzkt.TxsFormat{
				{
					To:      "tz1Wi5BHFA4qqr6cSXoQkpSKU8F1aMdL5cvs",
					Amount:  "1",
					TokenID: "37214540304218121786566893708923600581837527203284427749671447415338838815459",
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
	data := `[
		{
			"list":
			[
				{
					"to":       "tz1SKUGqfkVPe6xM5xkUiJwuSsmzswD2Rk41",
					"amount":   "1",
					"token_id": "7373"
				}
			],
			"address": "tz1dXJ4JuwTVq2uidgq6hxVgqvderghd1F5i"
		}
	]`
	var mapArray interface{}
	json.Unmarshal([]byte(data), &mapArray)

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
