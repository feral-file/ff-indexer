package worker

import "fmt"

func WorkflowIDIndexTokenProvenanceByIndexID(caller, indexID string) string {
	return fmt.Sprintf("index-token-provenance-by-helper-%s-%s", caller, indexID)
}

func WorkflowIDIndexTokenOwnershipByIndexID(caller, indexID string) string {
	return fmt.Sprintf("index-token-ownership-by-helper-%s-%s", caller, indexID)
}

func WorkflowIDIndexTokenByOwner(caller, owner string) string {
	return fmt.Sprintf("index-tokens-by-owner-%s-%s", caller, owner)
}

func WorkflowIDRefreshTokenProvenanceByOwner(caller, owner string) string {
	return fmt.Sprintf("refresh-tokens-provenance-by-owner-%s-%s", caller, owner)
}

func WorkflowIDIndexCollectionsByOwner(caller, owner string) string {
	return fmt.Sprintf("index-tokens-collections-by-owner-%s-%s", caller, owner)
}
