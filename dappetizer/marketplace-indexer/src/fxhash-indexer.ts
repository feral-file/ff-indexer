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
    FxhashAssignMetadataParameter,
    FxhashBalanceOfParameter,
    FxhashChangedStorage,
    FxhashInitialStorage,
    FxhashLedgerKey,
    FxhashLedgerValue,
    FxhashMetadataKey,
    FxhashMetadataValue,
    FxhashMintParameter,
    FxhashOperatorsKey,
    FxhashOperatorsValue,
    FxhashSetAdministratorParameter,
    FxhashSetIssuerParameter,
    FxhashSetMetadataParameter,
    FxhashSetPauseParameter,
    FxhashSetSignerParameter,
    FxhashSetTreasuryAddressParameter,
    FxhashTokenDataKey,
    FxhashTokenDataValue,
    FxhashTokenMetadataKey,
    FxhashTokenMetadataValue,
    FxhashTransferParameter,
    FxhashTransferXtzTreasuryParameter,
    FxhashUpdateOperatorsParameter,
} from './fxhash-indexer-interfaces.generated';

import { outputTransfer as outputTransfer } from './callback'

@contractFilter({ name: 'fxhash' })
export class FxhashIndexer {
    @indexOrigination()
    async indexOrigination(
        initialStorage: FxhashInitialStorage,
        dbContext: DbContext,
        indexingContext: OriginationIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('assign_metadata')
    async indexAssignMetadata(
        parameter: FxhashAssignMetadataParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('balance_of')
    async indexBalanceOf(
        parameter: FxhashBalanceOfParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('mint')
    async indexMint(
        parameter: FxhashMintParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_administrator')
    async indexSetAdministrator(
        parameter: FxhashSetAdministratorParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_issuer')
    async indexSetIssuer(
        parameter: FxhashSetIssuerParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_metadata')
    async indexSetMetadata(
        parameter: FxhashSetMetadataParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_pause')
    async indexSetPause(
        parameter: FxhashSetPauseParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_signer')
    async indexSetSigner(
        parameter: FxhashSetSignerParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_treasury_address')
    async indexSetTreasuryAddress(
        parameter: FxhashSetTreasuryAddressParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('transfer')
    async indexTransfer(
        parameter: FxhashTransferParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
        outputTransfer(parameter, indexingContext)
    }

    @indexEntrypoint('transfer_xtz_treasury')
    async indexTransferXtzTreasury(
        parameter: FxhashTransferXtzTreasuryParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('update_operators')
    async indexUpdateOperators(
        parameter: FxhashUpdateOperatorsParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexStorageChange()
    async indexStorageChange(
        newStorage: FxhashChangedStorage,
        dbContext: DbContext,
        indexingContext: StorageChangeIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['ledger'] })
    async indexLedgerUpdate(
        key: FxhashLedgerKey,
        value: FxhashLedgerValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['metadata'] })
    async indexMetadataUpdate(
        key: FxhashMetadataKey,
        value: FxhashMetadataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['operators'] })
    async indexOperatorsUpdate(
        key: FxhashOperatorsKey,
        value: FxhashOperatorsValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['token_data'] })
    async indexTokenDataUpdate(
        key: FxhashTokenDataKey,
        value: FxhashTokenDataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['token_metadata'] })
    async indexTokenMetadataUpdate(
        key: FxhashTokenMetadataKey,
        value: FxhashTokenMetadataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }
}
