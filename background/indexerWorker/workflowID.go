package indexerWorker

import "fmt"

func WorkflowIDIndexTokenProvenanceByOwner(address string) string {
	return fmt.Sprintf("index-token-provenance-by-owner-%s", address)
}

func WorkflowIDIndexTokenOwnershipByOwner(address string) string {
	return fmt.Sprintf("index-token-ownership-by-owner-%s", address)
}
