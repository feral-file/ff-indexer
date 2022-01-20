package indexer

const (
	LivenetZeroAddress = "a3ezwdYVEVrHwszQrYzDTCAZwUD3yKtNsCq9YhEu97bPaGAKy1"
	TestnetZeroAddress = "dw9MQXcC5rJZb3QE1nz86PiQAheMP1dx9M3dr52tT8NNs14m33"
)

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
