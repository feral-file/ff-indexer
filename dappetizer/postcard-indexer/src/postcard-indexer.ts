import BigNumber from 'bignumber.js';
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
    PostcardAddTrusteeParameter,
    PostcardAdminParameter,
    PostcardAssetsLedgerKey,
    PostcardAssetsLedgerValue,
    PostcardAssetsOperatorsKey,
    PostcardAssetsOperatorsValue,
    PostcardAssetsParameter,
    PostcardAssetsTokenMetadataKey,
    PostcardAssetsTokenMetadataValue,
    PostcardAssetsTokenTotalSupplyKey,
    PostcardAssetsTokenTotalSupplyValue,
    PostcardBalanceOfParameter,
    PostcardChangedStorage,
    PostcardConfirmAdminParameter,
    PostcardInitialStorage,
    PostcardMailPostcardParameter,
    PostcardMetadataKey,
    PostcardMetadataValue,
    PostcardMintPostcardParameter,
    PostcardPauseParameter,
    PostcardPostcardsKey,
    PostcardPostcardsParameter,
    PostcardPostcardsValue,
    PostcardRemoveTrusteeParameter,
    PostcardSetAdminParameter,
    PostcardStampPostcardParameter,
    PostcardTransferParameter,
    PostcardTrusteeParameter,
    PostcardUpdateOperatorsParameter,
} from './postcard-indexer-interfaces.generated';

import { outputTransfer, outputStampUpdate } from './callback'

@contractFilter({ name: 'postcard' })
export class PostcardIndexer {
    @indexOrigination()
    async indexOrigination(
        initialStorage: PostcardInitialStorage,
        dbContext: DbContext,
        indexingContext: OriginationIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('add_trustee')
    async indexAddTrustee(
        parameter: PostcardAddTrusteeParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('admin')
    async indexAdmin(
        parameter: PostcardAdminParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('assets')
    async indexAssets(
        parameter: PostcardAssetsParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('balance_of')
    async indexBalanceOf(
        parameter: PostcardBalanceOfParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('confirm_admin')
    async indexConfirmAdmin(
        parameter: PostcardConfirmAdminParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('mail_postcard')
    async indexMailPostcard(
        parameter: PostcardMailPostcardParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('mint_postcard')
    async indexMintPostcard(
        parameter: PostcardMintPostcardParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        parameter.forEach((p) => {
            outputTransfer([
                { from_: "", txs: [{ amount: BigNumber(1), to_: p.owner, token_id: p.token_id }] }
            ], indexingContext)
        })
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('pause')
    async indexPause(
        parameter: PostcardPauseParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('postcards')
    async indexPostcards(
        parameter: PostcardPostcardsParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('remove_trustee')
    async indexRemoveTrustee(
        parameter: PostcardRemoveTrusteeParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_admin')
    async indexSetAdmin(
        parameter: PostcardSetAdminParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('stamp_postcard')
    async indexStampPostcard(
        parameter: PostcardStampPostcardParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        outputStampUpdate(parameter, indexingContext)
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('transfer')
    async indexTransfer(
        parameter: PostcardTransferParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('trustee')
    async indexTrustee(
        parameter: PostcardTrusteeParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('update_operators')
    async indexUpdateOperators(
        parameter: PostcardUpdateOperatorsParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexStorageChange()
    async indexStorageChange(
        newStorage: PostcardChangedStorage,
        dbContext: DbContext,
        indexingContext: StorageChangeIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['assets', 'ledger'] })
    async indexAssetsLedgerUpdate(
        key: PostcardAssetsLedgerKey,
        value: PostcardAssetsLedgerValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['assets', 'operators'] })
    async indexAssetsOperatorsUpdate(
        key: PostcardAssetsOperatorsKey,
        value: PostcardAssetsOperatorsValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['assets', 'token_metadata'] })
    async indexAssetsTokenMetadataUpdate(
        key: PostcardAssetsTokenMetadataKey,
        value: PostcardAssetsTokenMetadataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['assets', 'token_total_supply'] })
    async indexAssetsTokenTotalSupplyUpdate(
        key: PostcardAssetsTokenTotalSupplyKey,
        value: PostcardAssetsTokenTotalSupplyValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['metadata'] })
    async indexMetadataUpdate(
        key: PostcardMetadataKey,
        value: PostcardMetadataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['postcards'] })
    async indexPostcardsUpdate(
        key: PostcardPostcardsKey,
        value: PostcardPostcardsValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }
}
