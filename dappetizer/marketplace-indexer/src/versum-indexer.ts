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
    Versum24Key,
    Versum24Value,
    VersumAddMigrationTargetParameter,
    VersumBalanceOfParameter,
    VersumChangedStorage,
    VersumDecreaseRoyaltiesParameter,
    VersumDisableNoTransfersUntilParameter,
    VersumDisableReqVerifiedToHoldParameter,
    VersumExtraDbKey,
    VersumExtraDbValue,
    VersumGenericParameter,
    VersumIncreaseMaxPerWalletParameter,
    VersumInitialStorage,
    VersumLedgerKey,
    VersumLedgerValue,
    VersumMetadataKey,
    VersumMetadataValue,
    VersumMigrateParameter,
    VersumMintParameter,
    VersumMintSlotsKey,
    VersumMintSlotsValue,
    VersumMutezTransferParameter,
    VersumOperatorsKey,
    VersumOperatorsValue,
    VersumPayRoyaltiesFa2Parameter,
    VersumPayRoyaltiesXtzParameter,
    VersumSetAdministratorParameter,
    VersumSetContractRegistryParameter,
    VersumSetContractsMhtParameter,
    VersumSetIdentityParameter,
    VersumSetMarketParameter,
    VersumSetMateriaAddressParameter,
    VersumSetMetadataParameter,
    VersumSetMintMateriaCostParameter,
    VersumSetMintingPausedParameter,
    VersumSetPauseParameter,
    VersumSetVerifiedToMintParameter,
    VersumSignCocreatorParameter,
    VersumSignedCoCreatorshipKey,
    VersumSignedCoCreatorshipValue,
    VersumTokenExtraDataKey,
    VersumTokenExtraDataValue,
    VersumTokenMetadataKey,
    VersumTokenMetadataValue,
    VersumTotalSupplyKey,
    VersumTotalSupplyValue,
    VersumTransferParameter,
    VersumUlAllContractsMhtParameter,
    VersumUnlockContractsMayHoldTokenParameter,
    VersumUpdateEpParameter,
    VersumUpdateExtraDbParameter,
    VersumUpdateMintSlotsParameter,
    VersumUpdateOperatorsParameter,
    VersumUpdateTokenMetadataParameter,
} from './versum-indexer-interfaces.generated';

import { outputTransfer } from './callback'

@contractFilter({ name: 'versum' })
export class VersumIndexer {
    @indexOrigination()
    async indexOrigination(
        initialStorage: VersumInitialStorage,
        dbContext: DbContext,
        indexingContext: OriginationIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('_add_migration_target')
    async indexAddMigrationTarget(
        parameter: VersumAddMigrationTargetParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('balance_of')
    async indexBalanceOf(
        parameter: VersumBalanceOfParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('decrease_royalties')
    async indexDecreaseRoyalties(
        parameter: VersumDecreaseRoyaltiesParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('disable_no_transfers_until')
    async indexDisableNoTransfersUntil(
        parameter: VersumDisableNoTransfersUntilParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('disable_req_verified_to_hold')
    async indexDisableReqVerifiedToHold(
        parameter: VersumDisableReqVerifiedToHoldParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('generic')
    async indexGeneric(
        parameter: VersumGenericParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('increase_max_per_wallet')
    async indexIncreaseMaxPerWallet(
        parameter: VersumIncreaseMaxPerWalletParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('migrate')
    async indexMigrate(
        parameter: VersumMigrateParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('mint')
    async indexMint(
        parameter: VersumMintParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('mutez_transfer')
    async indexMutezTransfer(
        parameter: VersumMutezTransferParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('pay_royalties_fa2')
    async indexPayRoyaltiesFa2(
        parameter: VersumPayRoyaltiesFa2Parameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('pay_royalties_xtz')
    async indexPayRoyaltiesXtz(
        parameter: VersumPayRoyaltiesXtzParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_administrator')
    async indexSetAdministrator(
        parameter: VersumSetAdministratorParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('_set_contract_registry')
    async indexSetContractRegistry(
        parameter: VersumSetContractRegistryParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('_set_contracts_mht')
    async indexSetContractsMht(
        parameter: VersumSetContractsMhtParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('_set_identity')
    async indexSetIdentity(
        parameter: VersumSetIdentityParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('_set_market')
    async indexSetMarket(
        parameter: VersumSetMarketParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('_set_materia_address')
    async indexSetMateriaAddress(
        parameter: VersumSetMateriaAddressParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_metadata')
    async indexSetMetadata(
        parameter: VersumSetMetadataParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('_set_mint_materia_cost')
    async indexSetMintMateriaCost(
        parameter: VersumSetMintMateriaCostParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('_set_minting_paused')
    async indexSetMintingPaused(
        parameter: VersumSetMintingPausedParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('set_pause')
    async indexSetPause(
        parameter: VersumSetPauseParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('_set_verified_to_mint')
    async indexSetVerifiedToMint(
        parameter: VersumSetVerifiedToMintParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('sign_cocreator')
    async indexSignCocreator(
        parameter: VersumSignCocreatorParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('transfer')
    async indexTransfer(
        parameter: VersumTransferParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
        outputTransfer(parameter, indexingContext)
    }

    @indexEntrypoint('_ul_all_contracts_mht')
    async indexUlAllContractsMht(
        parameter: VersumUlAllContractsMhtParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('unlock_contracts_may_hold_token')
    async indexUnlockContractsMayHoldToken(
        parameter: VersumUnlockContractsMayHoldTokenParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('update_ep')
    async indexUpdateEp(
        parameter: VersumUpdateEpParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('update_extra_db')
    async indexUpdateExtraDb(
        parameter: VersumUpdateExtraDbParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('_update_mint_slots')
    async indexUpdateMintSlots(
        parameter: VersumUpdateMintSlotsParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('update_operators')
    async indexUpdateOperators(
        parameter: VersumUpdateOperatorsParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexEntrypoint('update_token_metadata')
    async indexUpdateTokenMetadata(
        parameter: VersumUpdateTokenMetadataParameter,
        dbContext: DbContext,
        indexingContext: TransactionIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexStorageChange()
    async indexStorageChange(
        newStorage: VersumChangedStorage,
        dbContext: DbContext,
        indexingContext: StorageChangeIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['24'] })
    async index24Update(
        key: Versum24Key,
        value: Versum24Value | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['extra_db'] })
    async indexExtraDbUpdate(
        key: VersumExtraDbKey,
        value: VersumExtraDbValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['ledger'] })
    async indexLedgerUpdate(
        key: VersumLedgerKey,
        value: VersumLedgerValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['metadata'] })
    async indexMetadataUpdate(
        key: VersumMetadataKey,
        value: VersumMetadataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['mint_slots'] })
    async indexMintSlotsUpdate(
        key: VersumMintSlotsKey,
        value: VersumMintSlotsValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['operators'] })
    async indexOperatorsUpdate(
        key: VersumOperatorsKey,
        value: VersumOperatorsValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['signed_co_creatorship'] })
    async indexSignedCoCreatorshipUpdate(
        key: VersumSignedCoCreatorshipKey,
        value: VersumSignedCoCreatorshipValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['token_extra_data'] })
    async indexTokenExtraDataUpdate(
        key: VersumTokenExtraDataKey,
        value: VersumTokenExtraDataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['token_metadata'] })
    async indexTokenMetadataUpdate(
        key: VersumTokenMetadataKey,
        value: VersumTokenMetadataValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }

    @indexBigMapUpdate({ path: ['total_supply'] })
    async indexTotalSupplyUpdate(
        key: VersumTotalSupplyKey,
        value: VersumTotalSupplyValue | undefined, // Undefined represents a removal.
        dbContext: DbContext,
        indexingContext: BigMapUpdateIndexingContext,
    ): Promise<void> {
        // Implement your indexing logic here or delete the method if not needed.
    }
}
