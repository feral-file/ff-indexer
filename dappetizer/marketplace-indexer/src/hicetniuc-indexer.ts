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
    HicetniucBalanceOfParameter,
    HicetniucChangedStorage,
    HicetniucHDaoBatchParameter,
    HicetniucInitialStorage,
    HicetniucLedgerKey,
    HicetniucLedgerValue,
    HicetniucMetadataKey,
    HicetniucMetadataValue,
    HicetniucMintParameter,
    HicetniucOperatorsKey,
    HicetniucOperatorsValue,
    HicetniucSetAdministratorParameter,
    HicetniucSetPauseParameter,
    HicetniucTokenMetadataKey,
    HicetniucTokenMetadataParameter,
    HicetniucTokenMetadataValue,
    HicetniucTransferParameter,
    HicetniucUpdateOperatorsParameter,
} from './hicetniuc-indexer-interfaces.generated';

import { outputTransfer as outputTransfer } from './callback'

@contractFilter({ name: 'hicetniuc' })
export class HicetniucIndexer {
    @indexOrigination()
    async indexOrigination(
        initialStorage: HicetniucInitialStorage,
        dbContext: DbContext,
        indexingContext: OriginationIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('balance_of')
    async indexBalanceOf(
        parameter: HicetniucBalanceOfParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('hDAO_batch')
    async indexHDaoBatch(
        parameter: HicetniucHDaoBatchParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('mint')
    async indexMint(
        parameter: HicetniucMintParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_administrator')
    async indexSetAdministrator(
        parameter: HicetniucSetAdministratorParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_pause')
    async indexSetPause(
        parameter: HicetniucSetPauseParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('token_metadata')
    async indexTokenMetadata(
        parameter: HicetniucTokenMetadataParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('transfer')
    async indexTransfer(
        parameter: HicetniucTransferParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
        outputTransfer(parameter, indexingContext)
    }

    @indexEntrypoint('update_operators')
    async indexUpdateOperators(
        parameter: HicetniucUpdateOperatorsParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexStorageChange()
    async indexStorageChange(
        newStorage: HicetniucChangedStorage,
        dbContext: DbContext,
        indexingContext: StorageChangeIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['ledger'] })
    async indexLedgerUpdate(
        key: HicetniucLedgerKey,
        value: HicetniucLedgerValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['metadata'] })
    async indexMetadataUpdate(
        key: HicetniucMetadataKey,
        value: HicetniucMetadataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['operators'] })
    async indexOperatorsUpdate(
        key: HicetniucOperatorsKey,
        value: HicetniucOperatorsValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['token_metadata'] })
    async indexTokenMetadataUpdate(
        key: HicetniucTokenMetadataKey,
        value: HicetniucTokenMetadataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }
}
