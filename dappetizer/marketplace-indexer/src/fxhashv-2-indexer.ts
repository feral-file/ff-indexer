import { DbContext } from '@tezos-dappetizer/database';
import {
    contractFilter,
    indexBigMapUpdate,
    indexEntrypoint,
    indexOrigination,
    indexStorageChange,
} from '@tezos-dappetizer/decorators';
import {
    BigMapUpdateIndexingContext,
    OriginationIndexingContext,
    StorageChangeIndexingContext,
    TransactionIndexingContext,
} from '@tezos-dappetizer/indexer';

import {
    Fxhashv2AssignMetadataParameter,
    Fxhashv2BalanceOfParameter,
    Fxhashv2ChangedStorage,
    Fxhashv2InitialStorage,
    Fxhashv2LedgerKey,
    Fxhashv2LedgerValue,
    Fxhashv2MetadataKey,
    Fxhashv2MetadataValue,
    Fxhashv2MintParameter,
    Fxhashv2OperatorsKey,
    Fxhashv2OperatorsValue,
    Fxhashv2SetAdministratorParameter,
    Fxhashv2SetIssuerParameter,
    Fxhashv2SetMetadataParameter,
    Fxhashv2SetPauseParameter,
    Fxhashv2SetSignerParameter,
    Fxhashv2SetTreasuryAddressParameter,
    Fxhashv2TokenDataKey,
    Fxhashv2TokenDataValue,
    Fxhashv2TokenMetadataKey,
    Fxhashv2TokenMetadataValue,
    Fxhashv2TransferParameter,
    Fxhashv2TransferXtzTreasuryParameter,
    Fxhashv2UpdateOperatorsParameter,
} from './fxhashv-2-indexer-interfaces.generated';

import { outputTransfer as outputTransfer } from './callback'

@contractFilter({ name: 'fxhashv2' })
export class Fxhashv2Indexer {
    @indexOrigination()
    async indexOrigination(
        initialStorage: Fxhashv2InitialStorage,
        dbContext: DbContext,
        indexingContext: OriginationIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('assign_metadata')
    async indexAssignMetadata(
        parameter: Fxhashv2AssignMetadataParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('balance_of')
    async indexBalanceOf(
        parameter: Fxhashv2BalanceOfParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('mint')
    async indexMint(
        parameter: Fxhashv2MintParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_administrator')
    async indexSetAdministrator(
        parameter: Fxhashv2SetAdministratorParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_issuer')
    async indexSetIssuer(
        parameter: Fxhashv2SetIssuerParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_metadata')
    async indexSetMetadata(
        parameter: Fxhashv2SetMetadataParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_pause')
    async indexSetPause(
        parameter: Fxhashv2SetPauseParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_signer')
    async indexSetSigner(
        parameter: Fxhashv2SetSignerParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_treasury_address')
    async indexSetTreasuryAddress(
        parameter: Fxhashv2SetTreasuryAddressParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('transfer')
    async indexTransfer(
        parameter: Fxhashv2TransferParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
        outputTransfer(parameter, indexingContext)
    }

    @indexEntrypoint('transfer_xtz_treasury')
    async indexTransferXtzTreasury(
        parameter: Fxhashv2TransferXtzTreasuryParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('update_operators')
    async indexUpdateOperators(
        parameter: Fxhashv2UpdateOperatorsParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexStorageChange()
    async indexStorageChange(
        newStorage: Fxhashv2ChangedStorage,
        dbContext: DbContext,
        indexingContext: StorageChangeIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['ledger'] })
    async indexLedgerUpdate(
        key: Fxhashv2LedgerKey,
        value: Fxhashv2LedgerValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['metadata'] })
    async indexMetadataUpdate(
        key: Fxhashv2MetadataKey,
        value: Fxhashv2MetadataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['operators'] })
    async indexOperatorsUpdate(
        key: Fxhashv2OperatorsKey,
        value: Fxhashv2OperatorsValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['token_data'] })
    async indexTokenDataUpdate(
        key: Fxhashv2TokenDataKey,
        value: Fxhashv2TokenDataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['token_metadata'] })
    async indexTokenMetadataUpdate(
        key: Fxhashv2TokenMetadataKey,
        value: Fxhashv2TokenMetadataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }
}
