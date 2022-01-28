package indexer

const (
	LivenetZeroAddress = "a3ezwdYVEVrHwszQrYzDTCAZwUD3yKtNsCq9YhEu97bPaGAKy1"
	TestnetZeroAddress = "dw9MQXcC5rJZb3QE1nz86PiQAheMP1dx9M3dr52tT8NNs14m33"
)

const ENSContractAddress = "0x57f1887a8BF19b14fC0dF6Fd9B2acc9Af147eA85"
const KALAMContractAddress = "KT1A5P4ejnLix13jtadsfV9GCnXLMNnab8UT"

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
