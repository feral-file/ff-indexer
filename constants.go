package indexer

import utils "github.com/bitmark-inc/autonomy-utils"

const (
	LivenetZeroAddress = "a3ezwdYVEVrHwszQrYzDTCAZwUD3yKtNsCq9YhEu97bPaGAKy1"
	TestnetZeroAddress = "dw9MQXcC5rJZb3QE1nz86PiQAheMP1dx9M3dr52tT8NNs14m33"
)

// Ethereum contract addresses
// ignored contracts
const ENSContractAddress1 = "0x57f1887a8BF19b14fC0dF6Fd9B2acc9Af147eA85"
const ENSContractAddress2 = "0xD4416b13d2b3a9aBae7AcD5D6C2BbDBE25686401"

// index excluded addresses
const EthereumZeroAddress = "0x0000000000000000000000000000000000000000"
const EthereumDeadAddress = "0x000000000000000000000000000000000000dEaD"

var EthereumIndexExcludedOwners = map[string]struct{}{
	EthereumZeroAddress: {},
	EthereumDeadAddress: {},
}

const TransferEventSignature = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
const TransferSingleEventSignature = "0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62"

// Series Registry Contract
const SeriesRegistryEventRegisterSeriesSignature = "0x55d82c1e0fbf557aad06476685a2e64309e639e7b9763ffc3cffce16cb33f689"
const SeriesRegistryEventUpdateSeriesSignature = "0x799805152d6bdc2303ac0dbac07df8ecbd5946b3f66426ab316f5f3670876096"
const SeriesRegistryEventDeleteSeriesSignature = "0xd1cb5d9dfefdfe2dda1152786a83292f2b48e6c71981358d0f8b9eb7dd92d6f4"
const SeriesRegistryEventOptOutSeriesSignature = "0xea7a51899c6e1e1b64a73aba8ff9ce974a129f879f83343326f6c09d42834620"
const SeriesRegistryEventUpdateArtistAddressSignature = "0x4908f8df298e29e82b08d2367b89cc9e0a041d1fa94df05fb14872975cd56597"
const SeriesRegistryEventOptInCollaborationSignature = "0xf1be82cf03ce2483decce6c97c3580b7298a5d3ba618b98bc41e7c208007f22f"
const SeriesRegistryEventAssignSeriesSignature = "0x848e74caa538f5af05a1af3c186cca1e9d0c6f0c25995648face20eea1e930c7"

// Tezos contract addresses
// ignored contracts
const TezDaoContractAddress = "KT1C9X9s5rpVJGxwVuHEVBLYEdAQ1Qw8QDjH"
const TezosDNSContractAddress = "KT1GBZmSxmnKJXGMdMLbugPfLyUPmuLSMwKS"
const KALAMContractAddress = "KT1A5P4ejnLix13jtadsfV9GCnXLMNnab8UT"

// index excluded addresses
const (
	TezosNullAddress                 = "tz1Ke2h7sDdakHJQh8WX4Z372du1KChsksyU"
	TezosBurnAddress                 = "tz1burnburnburnburnburnburnburjAYjjX"
	TezosHicEtNuncMarketplaceAddress = "KT1HbQepzV1nVGg8QVznG7z4RcHseD5kwqBn"
	TezosTeiaMarketplaceAddress      = "KT1PHubm9HtyQEJ4BBpMTVomq6mhbfNZ9z5w"
	TezosOBJKTMarketplaceAddress     = "KT1FvqJwEDWb1Gwc55Jd1jjTHRVWbYKUUpyq"
	TezosOBJKTMarketplaceAddressV2   = "KT1WvzYHCNBvDSdwafTHv7nJ1dWmZ8GCYuuC"
	TezosOBJKTTreasuryProxyAddress   = "KT19kCpYFxrNPegMBYKKH44szaqv8offqBRz"
)

// testent
const (
	TezosOBJKTMarketplaceAddressTestnet = "KT1FvqJwEDWb1Gwc55Jd1jjTHRVWbYKUUpyq"
)

var TezosIndexExcludedOwners = map[string]struct{}{
	TezosNullAddress:                 {},
	TezosBurnAddress:                 {},
	TezosHicEtNuncMarketplaceAddress: {},
	TezosTeiaMarketplaceAddress:      {},
	TezosOBJKTMarketplaceAddress:     {},
}

// supported contract
const FXHASHContractAddressFX0_0 = "KT1U6EHmNxJTkvaWJ4ThczG4FSDaHC21ssvi"
const FXHASHContractAddressFX0_1 = "KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE"
const FXHASHContractAddressFX0_2 = "KT1AEVuykWeuuFX7QkEAMNtffzwhe1Z98hJS"
const FXHASHContractAddressFX1 = "KT1EfsNuqwLAWDd3o4pvfUx1CAh5GMdTrRvr"
const VersumContractAddress = "KT1LjmAdYQCLBjwv4S2oFkEzyHVkomAf5MrW"
const HicEtNuncContractAddress = "KT1RJ6PbjHpwc3M5rw5s2Nbmefwbuwbdxton"

// development contract
const FXHASHContractAddressDev0_0 = "KT1NkZho1yRkDdQnN4Mz93sDYyY2pPrEHTNs"
const FXHASHContractAddressDev0_1 = "KT1TtVAyjh4Ahdm8sLZwFnL7tqoLf59XrK2h"

var OBJKTSaleEntrypoints = []string{"fulfill_ask", "fulfill_offer"}

const (
	SourceFeralFile = "feralfile"
	SourceOpensea   = "opensea"
	SourceTZKT      = "tzkt"
)

var BlockchainAlias = map[string]string{
	utils.BitmarkBlockchain:  "bmk",
	utils.EthereumBlockchain: "eth",
	utils.TezosBlockchain:    "tez",
}

const (
	ObjktCDNDisplayType           = "display"
	ObjktCDNArtifactType          = "artifact"
	ObjktCDNThumbnailType         = "thumbnail"
	ObjktCDNArtifactThumbnailType = "thumb288"
	ObjktCDNBasePath              = "/file/assets-003/"
	ObjktCDNHost                  = "assets.objkt.media"
)

// ObjktCDNTypes should be in order, make sure ObjktCDNThumbnailType stand behind ObjktCDNArtifactType
var ObjktCDNTypes = []string{
	ObjktCDNDisplayType,
	ObjktCDNArtifactType,
	ObjktCDNThumbnailType,
}

const (
	DevelopmentEnvironment = "development"
	StagingEnvironment     = "staging"
	ProductionEnvironment  = "production"
)

const hicetnuncDefaultThumbnailURL = "ipfs://QmNrhZHUaEqxhyLfqoq1mtHSipkWHeT31LNHb1QEbDHgnc"

var SupportedCurrencyPairs = map[string]bool{
	"ETH-USD": true,
	"XTZ-USD": true,
}
