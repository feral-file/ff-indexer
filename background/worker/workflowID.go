package worker

import "fmt"

func WorkflowIDIndexTokenProvenanceByOwner(address string) string {
	return fmt.Sprintf("index-token-provenance-by-owner-%s", address)
}

func WorkflowIDIndexTokenOwnershipByOwner(address string) string {
	return fmt.Sprintf("index-token-ownership-by-owner-%s", address)
}

func WorkflowIDIndexTokenProvenanceByHelper(caller, indexID string) string {
	return fmt.Sprintf("index-token-provenance-by-helper-%s-%s", caller, indexID)
}

func WorkflowIDIndexTokenOwnershipByHelper(caller, indexID string) string {
	return fmt.Sprintf("index-token-ownership-by-helper-%s-%s", caller, indexID)
}

func WorkflowIDIndexTokenByOwner(caller, owner string) string {
	return fmt.Sprintf("index-tokens-by-owner-%s-%s", caller, owner)
}

func WorkflowIDRefreshTokenProvenanceByOwner(caller, owner string) string {
	return fmt.Sprintf("refresh-tokens-provenance-by-owner-%s-%s", caller, owner)
}
