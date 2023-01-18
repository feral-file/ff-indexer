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
    FeralFileV1AddTrusteeParameter,
    FeralFileV1AdminParameter,
    FeralFileV1AssetsLedgerKey,
    FeralFileV1AssetsLedgerValue,
    FeralFileV1AssetsOperatorsKey,
    FeralFileV1AssetsOperatorsValue,
    FeralFileV1AssetsParameter,
    FeralFileV1AssetsTokenMetadataKey,
    FeralFileV1AssetsTokenMetadataValue,
    FeralFileV1AuthorizedTransferParameter,
    FeralFileV1BalanceOfParameter,
    FeralFileV1BurnEditionsParameter,
    FeralFileV1BytesUtilsKey,
    FeralFileV1BytesUtilsValue,
    FeralFileV1ChangedStorage,
    FeralFileV1ConfirmAdminParameter,
    FeralFileV1InitialStorage,
    FeralFileV1MetadataKey,
    FeralFileV1MetadataValue,
    FeralFileV1MintEditionsParameter,
    FeralFileV1MinterParameter,
    FeralFileV1RegisterArtworksParameter,
    FeralFileV1RemoveTrusteeParameter,
    FeralFileV1SetAdminParameter,
    FeralFileV1TokenAttributeKey,
    FeralFileV1TokenAttributeValue,
    FeralFileV1TransferParameter,
    FeralFileV1TrusteeParameter,
    FeralFileV1UpdateEditionMetadataParameter,
    FeralFileV1UpdateOperatorsParameter,
} from './feral-file-v-1-indexer-interfaces.generated';

import { outputTransfer as outputTransfer } from './callback'

@contractFilter({ name: 'FeralFileV1' })
export class FeralFileV1Indexer {
    @indexOrigination()
    async indexOrigination(
        initialStorage: FeralFileV1InitialStorage,
        dbContext: DbContext,
        indexingContext: OriginationIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('add_trustee')
    async indexAddTrustee(
        parameter: FeralFileV1AddTrusteeParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('admin')
    async indexAdmin(
        parameter: FeralFileV1AdminParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('assets')
    async indexAssets(
        parameter: FeralFileV1AssetsParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('authorized_transfer')
    async indexAuthorizedTransfer(
        parameter: FeralFileV1AuthorizedTransferParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
        outputTransfer(parameter, indexingContext)
    }

    @indexEntrypoint('balance_of')
    async indexBalanceOf(
        parameter: FeralFileV1BalanceOfParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('burn_editions')
    async indexBurnEditions(
        parameter: FeralFileV1BurnEditionsParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('confirm_admin')
    async indexConfirmAdmin(
        parameter: FeralFileV1ConfirmAdminParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('mint_editions')
    async indexMintEditions(
        parameter: FeralFileV1MintEditionsParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('minter')
    async indexMinter(
        parameter: FeralFileV1MinterParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('register_artworks')
    async indexRegisterArtworks(
        parameter: FeralFileV1RegisterArtworksParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('remove_trustee')
    async indexRemoveTrustee(
        parameter: FeralFileV1RemoveTrusteeParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_admin')
    async indexSetAdmin(
        parameter: FeralFileV1SetAdminParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('transfer')
    async indexTransfer(
        parameter: FeralFileV1TransferParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
        outputTransfer(parameter, indexingContext)
    }

    @indexEntrypoint('trustee')
    async indexTrustee(
        parameter: FeralFileV1TrusteeParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('update_edition_metadata')
    async indexUpdateEditionMetadata(
        parameter: FeralFileV1UpdateEditionMetadataParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('update_operators')
    async indexUpdateOperators(
        parameter: FeralFileV1UpdateOperatorsParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexStorageChange()
    async indexStorageChange(
        newStorage: FeralFileV1ChangedStorage,
        dbContext: DbContext,
        indexingContext: StorageChangeIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['assets', 'ledger'] })
    async indexAssetsLedgerUpdate(
        key: FeralFileV1AssetsLedgerKey,
        value: FeralFileV1AssetsLedgerValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['assets', 'operators'] })
    async indexAssetsOperatorsUpdate(
        key: FeralFileV1AssetsOperatorsKey,
        value: FeralFileV1AssetsOperatorsValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['assets', 'token_metadata'] })
    async indexAssetsTokenMetadataUpdate(
        key: FeralFileV1AssetsTokenMetadataKey,
        value: FeralFileV1AssetsTokenMetadataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['bytes_utils'] })
    async indexBytesUtilsUpdate(
        key: FeralFileV1BytesUtilsKey,
        value: FeralFileV1BytesUtilsValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['metadata'] })
    async indexMetadataUpdate(
        key: FeralFileV1MetadataKey,
        value: FeralFileV1MetadataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['token_attribute'] })
    async indexTokenAttributeUpdate(
        key: FeralFileV1TokenAttributeKey,
        value: FeralFileV1TokenAttributeValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }
}
