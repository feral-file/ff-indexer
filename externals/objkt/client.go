package objkt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const OBJKT_QUERY = `
  query getObjktDetailed($tokenId: String!, $fa2: String!) {
	token(where: {token_id: {_eq: $tokenId}, fa_contract: {_eq: $fa2}}) {
	...TokenDefault
	__typename
	}
  }

  fragment TokenDefault on token {
	pk
	token_id
	artifact_uri
	description
	display_uri
	thumbnail_uri
	fa_contract
	rights
	royalties {
	  ...Royalties
	  __typename
	}
	supply
	timestamp
	name
	mime
	last_listed
	highest_offer
	lowest_ask
	flag
	fa {
	  ...Fa
	  __typename
	}
	creators {
	  ...CreatorDefault
	  __typename
	}
	attributes {
	  attribute {
		id
		name
		type
		value
		__typename
	  }
	  __typename
	}
	__typename
  }

  fragment Royalties on royalties {
	id
	amount
	decimals
	receiver_address
	__typename
  }

  fragment Fa on fa {
	active_auctions
	active_listing
	contract
	description
	name
	owners
	logo
	volume_24h
	volume_total
	website
	twitter
	items
	floor_price
	type
	collection_type
	creator_address
	collection_id
	path
	token_link
	short_name
	live
	editions
	collaborators {
	  ...Invitation
	  __typename
	}
	creator {
	  ...UserDefault
	  __typename
	}
	__typename
  }

  fragment Invitation on invitation {
	collaborator_address
	fa_contract
	id
	status
	timestamp
	update_timestamp
	fa {
	  ...FaLight
	  __typename
	}
	holder {
	  ...UserDefault
	  __typename
	}
	__typename
  }

  fragment FaLight on fa {
	contract
	description
	name
	owners
	logo
	volume_24h
	volume_total
	floor_price
	type
	collection_type
	collection_id
	path
	token_link
	short_name
	live
	__typename
  }

  fragment UserDefault on holder {
	address
	alias
	website
	twitter
	description
	tzdomain
	flag
	logo
	__typename
  }

  fragment CreatorDefault on token_creator {
	creator_address
	holder {
	  ...UserDefault
	  __typename
	}
	__typename
  }

  fragment TokenHolders on token {
	holders(where: {quantity: {_gt: "0"}}) {
	  ...TokenHolderDefault
	  __typename
	}
	__typename
  }

  fragment TokenHolderDefault on token_holder {
	holder_address
	quantity
	token_pk
	holder {
	  ...UserDefault
	  __typename
	}
	__typename
  }

  fragment EnglishAuctionDefault on obj_english_auction {
	id
	hash
	fa_contract
	price_increment
	reserve
	shares
	start_time
	status
	end_time
	timestamp
	token_pk
	update_level
	update_timestamp
	hash
	contract_version
	seller_address
	highest_bid
	extension_time
	highest_bidder_address
	currency {
	  ...CurrencyDefault
	  __typename
	}
	bids {
	  ...EnglishAuctionBidsDefault
	  __typename
	}
	token {
	  ...TokenDefault
	  __typename
	}
	__typename
  }

  fragment CurrencyDefault on currency {
	fa_contract
	id
	type
	__typename
  }

  fragment EnglishAuctionBidsDefault on obj_english_bid {
	amount
	bidder {
	  ...UserDefault
	  __typename
	}
	bidder_address
	id
	timestamp
	__typename
  }

  fragment DutchAuctionDefault on obj_dutch_auction {
	id
	hash
	amount
	amount_left
	fa_contract
	end_price
	start_price
	end_price
	end_time
	shares
	start_time
	status
	timestamp
	token_pk
	update_level
	update_timestamp
	contract_version
	hash
	seller_address
	seller {
	  ...UserDefault
	  __typename
	}
	__typename
  }

  fragment OfferDefault on obj_offer {
	id
	price
	shares
	status
	timestamp
	token_pk
	update_timestamp
	fa_contract
	contract_version
	currency {
	  ...CurrencyDefault
	  __typename
	}
	token {
	  ...TokenLight
	  __typename
	}
	buyer {
	  ...UserDefault
	  __typename
	}
	seller {
	  ...UserDefault
	  __typename
	}
	__typename
  }

  fragment TokenLight on token {
	pk
	token_id
	artifact_uri
	description
	display_uri
	thumbnail_uri
	fa_contract
	supply
	timestamp
	name
	mime
	flag
	last_listed
	creators {
	  ...CreatorDefault
	  __typename
	}
	fa {
	  ...Fa
	  __typename
	}
	__typename
  }

  fragment AskDefault on obj_ask {
	id
	amount
	amount_left
	price
	shares
	status
	timestamp
	token_pk
	update_timestamp
	fa_contract
	contract_version
	currency {
	  ...CurrencyDefault
	  __typename
	}
	seller {
	  ...UserDefault
	  __typename
	}
	__typename
  }

  fragment SwapDefault on hen_swap {
	id
	amount
	amount_left
	price
	royalties
	status
	timestamp
	token_pk
	seller {
	  ...UserDefault
	  __typename
	}
	__typename
  }
`

type ObjktAPI struct {
	client   *http.Client
	endpoint string
}

func New(objktEndpoint string) *ObjktAPI {
	return &ObjktAPI{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		endpoint: objktEndpoint,
	}
}

func NewGetObjktDetailedParams(tokenID, contract string) (io.Reader, error) {
	var params bytes.Buffer

	if err := json.NewEncoder(&params).Encode(
		map[string]interface{}{
			"operationName": "getObjktDetailed",
			"variables": map[string]string{
				"tokenId": tokenID,
				"fa2":     contract,
			},
			"query": OBJKT_QUERY,
		}); err != nil {
		return nil, err
	}

	return &params, nil
}

type ObjktContractDetails struct {
	Name           string `json:"name"`
	CreatorAddress string `json:"creator_address"`
}

type ObjktTokenDetails struct {
	TokenID  string    `json:"token_id"`
	MintedAt time.Time `json:"timestamp"`
	Supply   int64     `json:"supply"`

	Name         string `json:"name"`
	Description  string `json:"description"`
	MIMEType     string `json:"mime"`
	ArtifactURI  string `json:"artifact_uri"`
	DisplayURI   string `json:"display_uri"`
	ThumbnailURI string `json:"thumbnail_uri"`

	Contract ObjktContractDetails `json:"fa"`
}

type GetObjktDetailedResult struct {
	Data struct {
		Tokens []ObjktTokenDetails `json:"token"`
	} `json:"data"`
}

func (api *ObjktAPI) GetObjktDetailed(ctx context.Context, id, contract string) (ObjktTokenDetails, error) {

	reqBody, err := NewGetObjktDetailedParams(id, contract)
	if err != nil {
		return ObjktTokenDetails{}, err
	}

	// https://api2.objkt.com/v1/graphql
	req, err := http.NewRequest("POST", api.endpoint, reqBody)
	if err != nil {
		return ObjktTokenDetails{}, err
	}

	resp, err := api.client.Do(req)
	if err != nil {
		return ObjktTokenDetails{}, err
	}
	defer resp.Body.Close()

	var result GetObjktDetailedResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ObjktTokenDetails{}, err
	}

	if len(result.Data.Tokens) > 0 {
		return result.Data.Tokens[0], nil
	} else {
		return ObjktTokenDetails{}, fmt.Errorf("token not found")
	}
}
