package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type AssetContract struct {
	Address    string `json:"address"`
	SchemaName string `json:"schema_name"`
}

type Asset struct {
	TokenID       string        `json:"token_id"`
	Name          string        `json:"name"`
	AssetContract AssetContract `json:"asset_contract"`
}

type IndexerAsset struct {
	IndexID string `json:"indexID"`
}

func main() {
	var openseaIndexIDs []string
	var indexerIndexIDs []string
	var inOpenseaNotInIndexer []string
	var inIndexerNotInOpensea []string

	owner := os.Getenv("OWNER")
	fmt.Println("owner: ", owner)

	openseaIndexIDs = getOpenseaToken(owner)
	indexerIndexIDs = getIndexerToken(owner)

	for _, item := range openseaIndexIDs {
		if !contains(indexerIndexIDs, item) {
			inOpenseaNotInIndexer = append(inOpenseaNotInIndexer, item)
		}
	}

	for _, item := range indexerIndexIDs {
		if !contains(openseaIndexIDs, item) {
			inIndexerNotInOpensea = append(inIndexerNotInOpensea, item)
		}
	}

	fmt.Println("the number of token in opensea API: ", len(openseaIndexIDs))
	fmt.Println("the number of token in indexer API: ", len(indexerIndexIDs))
	fmt.Println("IndexID In Opensea Not In Indexer: ", inOpenseaNotInIndexer)
	fmt.Println("IndexID In Indexer Not In Opensea: ", inIndexerNotInOpensea)
}

func getIndexerToken(owner string) []string {
	var indexerIndexIDs []string

	v := url.Values{
		"size":   []string{"1000"},
		"offset": []string{"0"},
		"owner":  []string{owner},
	}

	u := url.URL{
		Scheme:   "https",
		Host:     "indexer.autonomy.io",
		Path:     "/nft",
		RawQuery: v.Encode(),
	}

	client := http.Client{}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		panic(err)
	}

	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	var indexerAssets []IndexerAsset

	if err := json.NewDecoder(res.Body).Decode(&indexerAssets); err != nil {
		panic(err)
	}

	for _, item := range indexerAssets {
		indexerIndexIDs = append(indexerIndexIDs, item.IndexID)
	}

	// map indexer checksum address to lowcase
	indexerIndexIDs = mapCheckSumAddressToLowcase(indexerIndexIDs, func(i string) string {
		return strings.ToLower(i)
	})

	return indexerIndexIDs
}

func getOpenseaToken(owner string) []string {
	var openseaIndexIDs []string
	offset := 0

	// get all indexID from Opensea API
	for {
		v := url.Values{
			"limit":           []string{"200"},
			"offset":          []string{fmt.Sprintf("%d", offset)},
			"order_direction": []string{"desc"},
			"owner":           []string{owner},
		}

		u := url.URL{
			Scheme:   "https",
			Host:     "api.opensea.io",
			Path:     "/api/v1/assets",
			RawQuery: v.Encode(),
		}

		client := http.Client{}
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			panic(err)
		}

		req.Header = http.Header{
			"X-API-KEY": {""},
		}

		res, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		var assetResp struct {
			Assets []Asset `json:"assets"`
		}

		if err := json.NewDecoder(res.Body).Decode(&assetResp); err != nil {
			panic(err)
		}

		for _, element := range assetResp.Assets {
			openseaIndexIDs = append(openseaIndexIDs, "eth-"+element.AssetContract.Address+"-"+element.TokenID)
		}

		if len(assetResp.Assets) < 200 {
			break
		}

		offset = offset + 200
	}

	return openseaIndexIDs
}

func mapCheckSumAddressToLowcase(data []string, f func(string) string) []string {
	mapped := make([]string, len(data))

	for i, e := range data {
		mapped[i] = f(e)
	}

	return mapped
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if e == a {
			return true
		}
	}
	return false
}
