package indexer

const (
	LivenetZeroAddress = "a3ezwdYVEVrHwszQrYzDTCAZwUD3yKtNsCq9YhEu97bPaGAKy1"
	TestnetZeroAddress = "dw9MQXcC5rJZb3QE1nz86PiQAheMP1dx9M3dr52tT8NNs14m33"
)

const EthereumZeroAddress = "0x0000000000000000000000000000000000000000"
const ENSContractAddress = "0x57f1887a8BF19b14fC0dF6Fd9B2acc9Af147eA85"

// Tezos contract addresses
const KALAMContractAddress = "KT1A5P4ejnLix13jtadsfV9GCnXLMNnab8UT"
const FXHASHContractAddress = "KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE"
const FXHASHV2ContractAddress = "KT1U6EHmNxJTkvaWJ4ThczG4FSDaHC21ssvi"
const FXHASHOldContractAddress = "KT1AEVuykWeuuFX7QkEAMNtffzwhe1Z98hJS"
const VersumContractAddress = "KT1LjmAdYQCLBjwv4S2oFkEzyHVkomAf5MrW"
const HicEtNuncContractAddress = "KT1RJ6PbjHpwc3M5rw5s2Nbmefwbuwbdxton"

const (
	BitmarkBlockchain  = "bitmark"
	EthereumBlockchain = "ethereum"
	TezosBlockchain    = "tezos"
)

var BlockchianAlias = map[string]string{
	BitmarkBlockchain:  "bmk",
	EthereumBlockchain: "eth",
	TezosBlockchain:    "tez",
}
