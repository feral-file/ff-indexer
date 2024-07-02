package etherscan

type TransactionQueryParams struct {
	Action     string  `structs:"action"`
	Address    *string `structs:"address,omitempty"`
	TxHash     *string `structs:"txhash,omitempty"`
	StartBlock *uint64 `structs:"startblock,omitempty"`
	EndBlock   *uint64 `structs:"endblock,omitempty"`
	Page       *uint64 `structs:"page,omitempty"`
	Offset     *uint64 `structs:"offset,omitempty"`
	Sort       *string `structs:"sort,omitempty"`
}

type Transaction struct {
	BlockNumber     string `json:"blockNumber"`
	Timestamp       string `json:"timeStamp"`
	Hash            string `json:"hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	ContractAddress string `json:"contractAddress"`
	Input           string `json:"input"`
	Type            string `json:"type"`
	Gas             string `json:"gas"`
	GasUsed         string `json:"gasUsed"`
	TraceID         string `json:"traceId"`
	IsError         string `json:"isError"`
	ErrorCode       string `json:"errCode"`
}
