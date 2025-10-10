package indexer

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/managedblockchainquery"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	log "github.com/bitmark-inc/autonomy-logger"
	"github.com/bitmark-inc/tzkt-go"

	"github.com/feral-file/ff-indexer/externals/fxhash"
	"github.com/feral-file/ff-indexer/externals/objkt"
	"github.com/feral-file/ff-indexer/externals/opensea"
)

func TestIPFSURLToGatewayURL(t *testing.T) {
	link := ipfsURLToGatewayURL(FxhashGateway, "ipfs://QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV?fxhash=opFymKKHJMKEGuYk5eafpiUS6PpYDKqVRqTUJqztG1F2N6nLVY")
	assert.Equal(t, "https://gateway.fxhash.xyz/ipfs/QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/?fxhash=opFymKKHJMKEGuYk5eafpiUS6PpYDKqVRqTUJqztG1F2N6nLVY", link)
	link = ipfsURLToGatewayURL(FxhashGateway, "ipfs://QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/?fxhash=opFymKKHJMKEGuYk5eafpiUS6PpYDKqVRqTUJqztG1F2N6nLVY")
	assert.Equal(t, "https://gateway.fxhash.xyz/ipfs/QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/?fxhash=opFymKKHJMKEGuYk5eafpiUS6PpYDKqVRqTUJqztG1F2N6nLVY", link)
	link = ipfsURLToGatewayURL(DefaultIPFSGateway, "ipfs://QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/01/index.html?a=1")
	assert.Equal(t, "https://ipfs.nftstorage.link/ipfs/QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/01/index.html?a=1", link)
	link = ipfsURLToGatewayURL(DefaultIPFSGateway, "ipfs://QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/01/?a=1")
	assert.Equal(t, "https://ipfs.nftstorage.link/ipfs/QmWKZY8Qp6U5WrC5Nxzf1xVaZoQgLzuDwbvDrnDdfjkBTV/01/?a=1", link)
}

func TestGetTokenBalanceOfOwner(t *testing.T) {
	if err := log.Initialize(false, nil); err != nil {
		panic(fmt.Errorf("fail to initialize logger with error: %s", err.Error()))
	}

	awsSession, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String("us-east-1"),
		},
	})
	if err != nil {
		log.Panic("fail to set up aws session", zap.Error(err))
	}

	blockchainQueryClient := managedblockchainquery.New(awsSession)

	engine := New(
		"",
		[]string{},
		map[string]string{},
		opensea.New("", 1),
		tzkt.New(""),
		fxhash.New("https://api.fxhash.xyz/graphql"),
		objkt.New(""),
		nil,
		nil,
		blockchainQueryClient,
	)

	balance1, err := engine.GetTokenBalanceOfOwner(context.Background(), "0xF903164aa2E070991467F7f9f0464d34B272013F", "22", "0x0F0eAE91990140C560D4156DB4f00c854Dc8F09E")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), balance1)

	balance2, err := engine.GetTokenBalanceOfOwner(context.Background(), "0x33FD426905F149f8376e227d0C9D3340AaD17aF1", "89", "0x0F0eAE91990140C560D4156DB4f00c854Dc8F09E")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), balance2)

	balance3, err := engine.GetTokenBalanceOfOwner(context.Background(), "0xe70659b717112ac4e14284d0db2f5d5703df8e43", "125", "0x0F0eAE91990140C560D4156DB4f00c854Dc8F09E")
	assert.NoError(t, err)
	assert.Equal(t, int64(4), balance3)
}

func TestOptimizedOpenseaImageURL(t *testing.T) {
	newURL1, err := OptimizedOpenseaImageURL("https://i.seadn.io/s/raw/files/aed53e1bcd90ca93b6fd3b0e012fadc5.jpg?w=500&auto=format")
	assert.NoError(t, err)
	assert.Equal(t, "https://i.seadn.io/s/raw/files/aed53e1bcd90ca93b6fd3b0e012fadc5.jpg?auto=format&dpr=1&w=3840", newURL1)

	newURL2, err := OptimizedOpenseaImageURL("https://i.seadn.io/gcs/files/0d06a393468f2e227ed14a6a88f951bc.jpg?w=500&auto=format")
	assert.NoError(t, err)
	assert.Equal(t, "https://i.seadn.io/gcs/files/0d06a393468f2e227ed14a6a88f951bc.jpg?auto=format&dpr=1&w=3840", newURL2)

	newURL3, err := OptimizedOpenseaImageURL("https://i.seadn.io/gae/zxrPTPWKa-uc-oLImHUN_bst5e6v7zeL5AIDXn1LWTpVe_43oCG2i-sZ5IsFHxHt4pkIuoDeaZF1HnApLqdy9wrzSeKSepRyYOr_9Q?w=500&auto=format")
	assert.NoError(t, err)
	assert.Equal(t, "https://i.seadn.io/gae/zxrPTPWKa-uc-oLImHUN_bst5e6v7zeL5AIDXn1LWTpVe_43oCG2i-sZ5IsFHxHt4pkIuoDeaZF1HnApLqdy9wrzSeKSepRyYOr_9Q?auto=format&dpr=1&w=3840", newURL3)
}

func TestGetEditionNumberByName(t *testing.T) {
	engine := &IndexEngine{}
	n1 := engine.GetEditionNumberByName("test 123")
	assert.Equal(t, int64(0), n1)
	n2 := engine.GetEditionNumberByName("test #123")
	assert.Equal(t, int64(123), n2)
}

func TestLookupArtistName(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]interface{}
		expected string
	}{
		{
			name: "artist field - direct match",
			metadata: map[string]interface{}{
				"artist": "John Doe",
			},
			expected: "John Doe",
		},
		{
			name: "traits field - artist trait",
			metadata: map[string]interface{}{
				"traits": []interface{}{
					map[string]interface{}{
						"trait_type": "artist",
						"value":      "Jane Smith",
					},
				},
			},
			expected: "Jane Smith",
		},
		{
			name: "traits field - Artist trait (capitalized)",
			metadata: map[string]interface{}{
				"traits": []interface{}{
					map[string]interface{}{
						"trait_type": "Artist",
						"value":      "Bob Wilson",
					},
				},
			},
			expected: "Bob Wilson",
		},
		{
			name: "traits field - Creator trait",
			metadata: map[string]interface{}{
				"traits": []interface{}{
					map[string]interface{}{
						"trait_type": "Creator",
						"value":      "Alice Brown",
					},
				},
			},
			expected: "Alice Brown",
		},
		{
			name: "traits field - creator trait (lowercase)",
			metadata: map[string]interface{}{
				"traits": []interface{}{
					map[string]interface{}{
						"trait_type": "creator",
						"value":      "Charlie Davis",
					},
				},
			},
			expected: "Charlie Davis",
		},
		{
			name: "traits field - multiple traits, artist first",
			metadata: map[string]interface{}{
				"traits": []interface{}{
					map[string]interface{}{
						"trait_type": "color",
						"value":      "blue",
					},
					map[string]interface{}{
						"trait_type": "artist",
						"value":      "David Lee",
					},
					map[string]interface{}{
						"trait_type": "size",
						"value":      "large",
					},
				},
			},
			expected: "David Lee",
		},
		{
			name: "traits field - invalid trait type",
			metadata: map[string]interface{}{
				"traits": []interface{}{
					map[string]interface{}{
						"trait_type": "invalid",
						"value":      "Some Value",
					},
				},
			},
			expected: "",
		},
		{
			name: "traits field - non-string trait type",
			metadata: map[string]interface{}{
				"traits": []interface{}{
					map[string]interface{}{
						"trait_type": 123,
						"value":      "Some Value",
					},
				},
			},
			expected: "",
		},
		{
			name: "traits field - non-string value",
			metadata: map[string]interface{}{
				"traits": []interface{}{
					map[string]interface{}{
						"trait_type": "artist",
						"value":      123,
					},
				},
			},
			expected: "",
		},
		{
			name: "traits field - non-map trait",
			metadata: map[string]interface{}{
				"traits": []interface{}{
					"invalid trait",
					map[string]interface{}{
						"trait_type": "artist",
						"value":      "Eve Johnson",
					},
				},
			},
			expected: "Eve Johnson",
		},
		{
			name: "collection_name field - with 'by' separator",
			metadata: map[string]interface{}{
				"collection_name": "Amazing Collection by Frank Miller",
			},
			expected: "Frank Miller",
		},
		{
			name: "collection_name field - empty string",
			metadata: map[string]interface{}{
				"collection_name": "",
			},
			expected: "",
		},
		{
			name: "collection_name field - no 'by' separator",
			metadata: map[string]interface{}{
				"collection_name": "Amazing Collection",
			},
			expected: "",
		},
		{
			name: "createdBy field",
			metadata: map[string]interface{}{
				"createdBy": "Grace Wilson",
			},
			expected: "Grace Wilson",
		},
		{
			name: "created_by field",
			metadata: map[string]interface{}{
				"created_by": "Henry Taylor",
			},
			expected: "Henry Taylor",
		},
		{
			name: "creator field",
			metadata: map[string]interface{}{
				"creator": "Ivy Chen",
			},
			expected: "Ivy Chen",
		},
		{
			name: "priority order - artist field takes precedence",
			metadata: map[string]interface{}{
				"artist":          "Primary Artist",
				"createdBy":       "Secondary Artist",
				"created_by":      "Tertiary Artist",
				"creator":         "Quaternary Artist",
				"collection_name": "Collection by Fifth Artist",
				"traits": []interface{}{
					map[string]interface{}{
						"trait_type": "artist",
						"value":      "Trait Artist",
					},
				},
			},
			expected: "Primary Artist",
		},
		{
			name: "priority order - traits take precedence over collection_name",
			metadata: map[string]interface{}{
				"createdBy":       "Secondary Artist",
				"created_by":      "Tertiary Artist",
				"creator":         "Quaternary Artist",
				"collection_name": "Collection by Fifth Artist",
				"traits": []interface{}{
					map[string]interface{}{
						"trait_type": "artist",
						"value":      "Trait Artist",
					},
				},
			},
			expected: "Trait Artist",
		},
		{
			name: "priority order - collection_name takes precedence over createdBy",
			metadata: map[string]interface{}{
				"createdBy":       "Secondary Artist",
				"created_by":      "Tertiary Artist",
				"creator":         "Quaternary Artist",
				"collection_name": "Collection by Fifth Artist",
			},
			expected: "Fifth Artist",
		},
		{
			name: "priority order - createdBy takes precedence over created_by",
			metadata: map[string]interface{}{
				"createdBy":  "Secondary Artist",
				"created_by": "Tertiary Artist",
				"creator":    "Quaternary Artist",
			},
			expected: "Secondary Artist",
		},
		{
			name: "priority order - created_by takes precedence over creator",
			metadata: map[string]interface{}{
				"created_by": "Tertiary Artist",
				"creator":    "Quaternary Artist",
			},
			expected: "Tertiary Artist",
		},
		{
			name:     "empty metadata",
			metadata: map[string]interface{}{},
			expected: "",
		},
		{
			name:     "nil metadata",
			metadata: nil,
			expected: "",
		},
		{
			name: "non-string artist field",
			metadata: map[string]interface{}{
				"artist": 123,
			},
			expected: "",
		},
		{
			name: "non-array traits field",
			metadata: map[string]interface{}{
				"traits": "not an array",
			},
			expected: "",
		},
		{
			name: "non-string collection_name field",
			metadata: map[string]interface{}{
				"collection_name": 123,
			},
			expected: "",
		},
		{
			name: "non-string createdBy field",
			metadata: map[string]interface{}{
				"createdBy": 123,
			},
			expected: "",
		},
		{
			name: "non-string created_by field",
			metadata: map[string]interface{}{
				"created_by": 123,
			},
			expected: "",
		},
		{
			name: "non-string creator field",
			metadata: map[string]interface{}{
				"creator": 123,
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lookupArtistName(tt.metadata)
			if result != tt.expected {
				t.Errorf("lookupArtistName() = %v, want %v", result, tt.expected)
			}
		})
	}
}
