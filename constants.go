package indexer

const (
	LivenetZeroAddress = "a3ezwdYVEVrHwszQrYzDTCAZwUD3yKtNsCq9YhEu97bPaGAKy1"
	TestnetZeroAddress = "dw9MQXcC5rJZb3QE1nz86PiQAheMP1dx9M3dr52tT8NNs14m33"
)

const EthereumZeroAddress = "0x0000000000000000000000000000000000000000"
const ENSContractAddress = "0x57f1887a8BF19b14fC0dF6Fd9B2acc9Af147eA85"
const TransferEventSignature = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
const TransferSingleEventSignature = "0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62"

// Tezos contract addresses
// ignored contracts
const TezDaoContractAddress = "KT1C9X9s5rpVJGxwVuHEVBLYEdAQ1Qw8QDjH"
const TezosDNSContractAddress = "KT1GBZmSxmnKJXGMdMLbugPfLyUPmuLSMwKS"
const KALAMContractAddress = "KT1A5P4ejnLix13jtadsfV9GCnXLMNnab8UT"

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
const POSTCARDCONTRACT = "KT1ESGez4dEuDjjNt4k2HPAK5Nzh7e8X8jyX"

const (
	SourceFeralFile = "feralfile"
	SourceOpensea   = "opensea"
	SourceTZKT      = "tzkt"
)

const (
	BitmarkBlockchain  = "bitmark"
	EthereumBlockchain = "ethereum"
	TezosBlockchain    = "tezos"
	UnknownBlockchain  = ""
)

var BlockchainAlias = map[string]string{
	BitmarkBlockchain:  "bmk",
	EthereumBlockchain: "eth",
	TezosBlockchain:    "tez",
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
	ProductiontEnvironment = "production"
)

const hicetnuncDefaultThumbnailURL = "ipfs://QmNrhZHUaEqxhyLfqoq1mtHSipkWHeT31LNHb1QEbDHgnc"
