/* istanbul ignore next */
/* eslint-disable */

// This file was generated.
// It should NOT be modified manually rather it should be regenerated.
// Contract: KT1LjmAdYQCLBjwv4S2oFkEzyHVkomAf5MrW
// Tezos network: mainnet

import { UnitValue } from '@taquito/michelson-encoder';
import { BigMapAbstraction, MichelsonMap } from '@taquito/taquito';
import { BigNumber } from 'bignumber.js';

export type VersumParameter =
    | { entrypoint: '_add_migration_target'; value: VersumAddMigrationTargetParameter }
    | { entrypoint: 'balance_of'; value: VersumBalanceOfParameter }
    | { entrypoint: 'decrease_royalties'; value: VersumDecreaseRoyaltiesParameter }
    | { entrypoint: 'disable_no_transfers_until'; value: VersumDisableNoTransfersUntilParameter }
    | { entrypoint: 'disable_req_verified_to_hold'; value: VersumDisableReqVerifiedToHoldParameter }
    | { entrypoint: 'generic'; value: VersumGenericParameter }
    | { entrypoint: 'increase_max_per_wallet'; value: VersumIncreaseMaxPerWalletParameter }
    | { entrypoint: 'migrate'; value: VersumMigrateParameter }
    | { entrypoint: 'mint'; value: VersumMintParameter }
    | { entrypoint: 'mutez_transfer'; value: VersumMutezTransferParameter }
    | { entrypoint: 'pay_royalties_fa2'; value: VersumPayRoyaltiesFa2Parameter }
    | { entrypoint: 'pay_royalties_xtz'; value: VersumPayRoyaltiesXtzParameter }
    | { entrypoint: 'set_administrator'; value: VersumSetAdministratorParameter }
    | { entrypoint: '_set_contract_registry'; value: VersumSetContractRegistryParameter }
    | { entrypoint: '_set_contracts_mht'; value: VersumSetContractsMhtParameter }
    | { entrypoint: '_set_identity'; value: VersumSetIdentityParameter }
    | { entrypoint: '_set_market'; value: VersumSetMarketParameter }
    | { entrypoint: '_set_materia_address'; value: VersumSetMateriaAddressParameter }
    | { entrypoint: 'set_metadata'; value: VersumSetMetadataParameter }
    | { entrypoint: '_set_mint_materia_cost'; value: VersumSetMintMateriaCostParameter }
    | { entrypoint: '_set_minting_paused'; value: VersumSetMintingPausedParameter }
    | { entrypoint: 'set_pause'; value: VersumSetPauseParameter }
    | { entrypoint: '_set_verified_to_mint'; value: VersumSetVerifiedToMintParameter }
    | { entrypoint: 'sign_cocreator'; value: VersumSignCocreatorParameter }
    | { entrypoint: 'transfer'; value: VersumTransferParameter }
    | { entrypoint: '_ul_all_contracts_mht'; value: VersumUlAllContractsMhtParameter }
    | { entrypoint: 'unlock_contracts_may_hold_token'; value: VersumUnlockContractsMayHoldTokenParameter }
    | { entrypoint: 'update_ep'; value: VersumUpdateEpParameter }
    | { entrypoint: 'update_extra_db'; value: VersumUpdateExtraDbParameter }
    | { entrypoint: '_update_mint_slots'; value: VersumUpdateMintSlotsParameter }
    | { entrypoint: 'update_operators'; value: VersumUpdateOperatorsParameter }
    | { entrypoint: 'update_token_metadata'; value: VersumUpdateTokenMetadataParameter };

export interface VersumAddMigrationTargetParameter {
    /** Bytes. */
    extra_data: string;

    /** Tezos address. */
    new_contract: string;
}

export interface VersumBalanceOfParameter {
    /** Contract address. */
    callback: string;

    requests: VersumBalanceOfParameterRequestsItem[];
}

export interface VersumBalanceOfParameterRequestsItem {
    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumDecreaseRoyaltiesParameter {
    /** Nat - arbitrary big integer >= 0. */
    new_royalties: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** Nat - arbitrary big integer >= 0. */
export type VersumDisableNoTransfersUntilParameter = BigNumber;

/** Nat - arbitrary big integer >= 0. */
export type VersumDisableReqVerifiedToHoldParameter = BigNumber;

/** Bytes. */
export type VersumGenericParameter = string;

export interface VersumIncreaseMaxPerWalletParameter {
    /** Nat - arbitrary big integer >= 0. */
    new_max: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumMigrateParameter {
    /** Tezos address. */
    '_from': string;

    /** Bytes. */
    extra_data: string;

    migrations: VersumMigrateParameterMigrationsItem[];

    /** Tezos address. */
    new_contract: string;
}

export interface VersumMigrateParameterMigrationsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumMintParameter {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    infusions: VersumMintParameterInfusionsItem[];

    /** Nat - arbitrary big integer >= 0. */
    max_per_address: BigNumber;

    /**
     * In-memory map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    metadata: MichelsonMap<string, string>;

    /** Date ISO 8601 string. */
    no_transfers_until: { Some: string } | null;

    /** Simple boolean. */
    req_verified_to_own: boolean;

    /** Nat - arbitrary big integer >= 0. */
    royalty: BigNumber;

    splits: VersumMintParameterSplitsItem[];
}

export interface VersumMintParameterInfusionsItem {
    /** Tezos address. */
    token_address: string;

    token_id_amounts: VersumMintParameterInfusionsItemTokenIdAmountsItem[];
}

export interface VersumMintParameterInfusionsItemTokenIdAmountsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumMintParameterSplitsItem {
    /** Tezos address. */
    address: string;

    /** Nat - arbitrary big integer >= 0. */
    pct: BigNumber;
}

export interface VersumMutezTransferParameter {
    /** Mutez - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Tezos address. */
    destination: string;
}

export interface VersumPayRoyaltiesFa2Parameter {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Tezos address. */
    buyer: string;

    fa2: VersumPayRoyaltiesFa2ParameterFa2;

    /** Tezos address. */
    seller: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumPayRoyaltiesFa2ParameterFa2 {
    /** Tezos address. */
    '3': string;

    /** Nat - arbitrary big integer >= 0. */
    '4': BigNumber;
}

export interface VersumPayRoyaltiesXtzParameter {
    /** Tezos address. */
    buyer: string;

    /** Tezos address. */
    seller: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** Tezos address. */
export type VersumSetAdministratorParameter = string;

/** Tezos address. */
export type VersumSetContractRegistryParameter = string;

/** Simple boolean. */
export type VersumSetContractsMhtParameter = boolean;

/** Tezos address. */
export type VersumSetIdentityParameter = string;

/** Tezos address. */
export type VersumSetMarketParameter = string;

/** Tezos address. */
export type VersumSetMateriaAddressParameter = string;

export interface VersumSetMetadataParameter {
    /** Arbitrary string. */
    k: string;

    /** Bytes. */
    v: string;
}

/** Nat - arbitrary big integer >= 0. */
export type VersumSetMintMateriaCostParameter = BigNumber;

/** Simple boolean. */
export type VersumSetMintingPausedParameter = boolean;

/** Simple boolean. */
export type VersumSetPauseParameter = boolean;

/** Simple boolean. */
export type VersumSetVerifiedToMintParameter = boolean;

/** Nat - arbitrary big integer >= 0. */
export type VersumSignCocreatorParameter = BigNumber;

export type VersumTransferParameter = VersumTransferParameterItem[];

export interface VersumTransferParameterItem {
    /** Tezos address. */
    from_: string;

    txs: VersumTransferParameterItemTxsItem[];
}

export interface VersumTransferParameterItemTxsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Tezos address. */
    to_: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumUlAllContractsMhtParameter {
    /** Tezos address. */
    minter: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** Nat - arbitrary big integer >= 0. */
export type VersumUnlockContractsMayHoldTokenParameter = BigNumber;

export interface VersumUpdateEpParameter {
    ep_name: VersumUpdateEpParameterEpName;

    /** Lambda. */
    new_ep: unknown;
}

export interface VersumUpdateEpParameterEpName {
    /** An empty result. */
    '_add_migration_target'?: typeof UnitValue;

    /** An empty result. */
    generic?: typeof UnitValue;

    /** An empty result. */
    migrate?: typeof UnitValue;

    /** An empty result. */
    mint?: typeof UnitValue;

    /** An empty result. */
    pay_royalties_fa2?: typeof UnitValue;

    /** An empty result. */
    pay_royalties_xtz?: typeof UnitValue;

    /** An empty result. */
    transfer?: typeof UnitValue;

    /** An empty result. */
    update_operators?: typeof UnitValue;
}

export interface VersumUpdateExtraDbParameter {
    /** Bytes. */
    key: string;

    /** Bytes. */
    value: string;
}

export interface VersumUpdateMintSlotsParameter {
    /** Simple boolean. */
    '_add': boolean;

    updates: VersumUpdateMintSlotsParameterUpdatesItem[];
}

export interface VersumUpdateMintSlotsParameterUpdatesItem {
    /** Simple boolean. */
    genesis_set: boolean;

    /** Tezos address. */
    minter: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export type VersumUpdateOperatorsParameter = VersumUpdateOperatorsParameterItem[];

export interface VersumUpdateOperatorsParameterItem {
    add_operator?: VersumUpdateOperatorsParameterItemAddOperator;

    remove_operator?: VersumUpdateOperatorsParameterItemRemoveOperator;
}

export interface VersumUpdateOperatorsParameterItemAddOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumUpdateOperatorsParameterItemRemoveOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumUpdateTokenMetadataParameter {
    /**
     * In-memory map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    metadata: MichelsonMap<string, string>;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumCurrentStorage {
    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `unknown`: Lambda.
     */
    '24': BigMapAbstraction;

    /** Lambda. */
    admin_check_lambda: unknown;

    /** Tezos address. */
    administrator: string;

    /** Array of: Nat - arbitrary big integer >= 0. */
    all_tokens: BigNumber[];

    /** Tezos address. */
    contract_registry: string;

    /** Simple boolean. */
    contracts_may_hold_tokens_global: boolean;

    /**
     * Big map.
     * 
     * Key of `string`: Bytes.
     * 
     * Value of `string`: Bytes.
     */
    extra_db: BigMapAbstraction;

    /** Array of: Tezos address. */
    genesis_minters: string[];

    /** Date ISO 8601 string. */
    genesis_timeout: string;

    /** Tezos address. */
    identity: string;

    /**
     * Big map.
     * 
     * Key of `VersumCurrentStorageLedgerKey`.
     * 
     * Value of `BigNumber`: Nat - arbitrary big integer >= 0.
     */
    ledger: BigMapAbstraction;

    /** Tezos address. */
    market: string;

    /** Tezos address. */
    materia_address: string;

    /**
     * Big map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    metadata: BigMapAbstraction;

    /** Nat - arbitrary big integer >= 0. */
    mint_materia_cost: BigNumber;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `string`: Tezos address.
     */
    mint_slots: BigMapAbstraction;

    /** Simple boolean. */
    minting_paused: boolean;

    /**
     * Big map.
     * 
     * Key of `VersumCurrentStorageOperatorsKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    operators: BigMapAbstraction;

    /** Simple boolean. */
    paused: boolean;

    /** Simple boolean. */
    require_verified_to_mint: boolean;

    /**
     * Big map.
     * 
     * Key of `VersumCurrentStorageSignedCoCreatorshipKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    signed_co_creatorship: BigMapAbstraction;

    /** Nat - arbitrary big integer >= 0. */
    token_counter: BigNumber;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `VersumCurrentStorageTokenExtraDataValue`.
     */
    token_extra_data: BigMapAbstraction;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `VersumCurrentStorageTokenMetadataValue`.
     */
    token_metadata: BigMapAbstraction;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `BigNumber`: Nat - arbitrary big integer >= 0.
     */
    total_supply: BigMapAbstraction;
}

export interface VersumCurrentStorageLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

export interface VersumCurrentStorageOperatorsKey {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumCurrentStorageSignedCoCreatorshipKey {
    /** Tezos address. */
    cocreator: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumCurrentStorageTokenExtraDataValue {
    /** Simple boolean. */
    contracts_may_hold_token: boolean;

    /**
     * In-memory map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    extra_fields: MichelsonMap<string, string>;

    infusions: VersumCurrentStorageTokenExtraDataValueInfusionsItem[];

    /** Nat - arbitrary big integer >= 0. */
    max_per_address: BigNumber;

    /** Tezos address. */
    minter: string;

    /** Date ISO 8601 string. */
    no_transfers_until: { Some: string } | null;

    /** Simple boolean. */
    req_verified_to_own: boolean;

    /** Nat - arbitrary big integer >= 0. */
    royalty: BigNumber;

    splits: VersumCurrentStorageTokenExtraDataValueSplitsItem[];
}

export interface VersumCurrentStorageTokenExtraDataValueInfusionsItem {
    /** Tezos address. */
    token_address: string;

    token_id_amounts: VersumCurrentStorageTokenExtraDataValueInfusionsItemTokenIdAmountsItem[];
}

export interface VersumCurrentStorageTokenExtraDataValueInfusionsItemTokenIdAmountsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumCurrentStorageTokenExtraDataValueSplitsItem {
    /** Tezos address. */
    address: string;

    /** Nat - arbitrary big integer >= 0. */
    pct: BigNumber;
}

export interface VersumCurrentStorageTokenMetadataValue {
    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;

    /**
     * In-memory map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    token_info: MichelsonMap<string, string>;
}

export interface VersumChangedStorage {
    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    '24': string;

    /** Lambda. */
    admin_check_lambda: unknown;

    /** Tezos address. */
    administrator: string;

    /** Array of: Nat - arbitrary big integer >= 0. */
    all_tokens: BigNumber[];

    /** Tezos address. */
    contract_registry: string;

    /** Simple boolean. */
    contracts_may_hold_tokens_global: boolean;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    extra_db: string;

    /** Array of: Tezos address. */
    genesis_minters: string[];

    /** Date ISO 8601 string. */
    genesis_timeout: string;

    /** Tezos address. */
    identity: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    ledger: string;

    /** Tezos address. */
    market: string;

    /** Tezos address. */
    materia_address: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    metadata: string;

    /** Nat - arbitrary big integer >= 0. */
    mint_materia_cost: BigNumber;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    mint_slots: string;

    /** Simple boolean. */
    minting_paused: boolean;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    operators: string;

    /** Simple boolean. */
    paused: boolean;

    /** Simple boolean. */
    require_verified_to_mint: boolean;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    signed_co_creatorship: string;

    /** Nat - arbitrary big integer >= 0. */
    token_counter: BigNumber;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    token_extra_data: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    token_metadata: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    total_supply: string;
}

export interface VersumInitialStorage {
    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `unknown`: Lambda.
     */
    '24': MichelsonMap<BigNumber, unknown>;

    /** Lambda. */
    admin_check_lambda: unknown;

    /** Tezos address. */
    administrator: string;

    /** Array of: Nat - arbitrary big integer >= 0. */
    all_tokens: BigNumber[];

    /** Tezos address. */
    contract_registry: string;

    /** Simple boolean. */
    contracts_may_hold_tokens_global: boolean;

    /**
     * Big map initial values.
     * 
     * Key of `string`: Bytes.
     * 
     * Value of `string`: Bytes.
     */
    extra_db: MichelsonMap<string, string>;

    /** Array of: Tezos address. */
    genesis_minters: string[];

    /** Date ISO 8601 string. */
    genesis_timeout: string;

    /** Tezos address. */
    identity: string;

    /**
     * Big map initial values.
     * 
     * Key of `VersumInitialStorageLedgerKey`.
     * 
     * Value of `BigNumber`: Nat - arbitrary big integer >= 0.
     */
    ledger: MichelsonMap<VersumInitialStorageLedgerKey, BigNumber>;

    /** Tezos address. */
    market: string;

    /** Tezos address. */
    materia_address: string;

    /**
     * Big map initial values.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    metadata: MichelsonMap<string, string>;

    /** Nat - arbitrary big integer >= 0. */
    mint_materia_cost: BigNumber;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `string`: Tezos address.
     */
    mint_slots: MichelsonMap<BigNumber, string>;

    /** Simple boolean. */
    minting_paused: boolean;

    /**
     * Big map initial values.
     * 
     * Key of `VersumInitialStorageOperatorsKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    operators: MichelsonMap<VersumInitialStorageOperatorsKey, typeof UnitValue>;

    /** Simple boolean. */
    paused: boolean;

    /** Simple boolean. */
    require_verified_to_mint: boolean;

    /**
     * Big map initial values.
     * 
     * Key of `VersumInitialStorageSignedCoCreatorshipKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    signed_co_creatorship: MichelsonMap<VersumInitialStorageSignedCoCreatorshipKey, typeof UnitValue>;

    /** Nat - arbitrary big integer >= 0. */
    token_counter: BigNumber;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `VersumInitialStorageTokenExtraDataValue`.
     */
    token_extra_data: MichelsonMap<BigNumber, VersumInitialStorageTokenExtraDataValue>;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `VersumInitialStorageTokenMetadataValue`.
     */
    token_metadata: MichelsonMap<BigNumber, VersumInitialStorageTokenMetadataValue>;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `BigNumber`: Nat - arbitrary big integer >= 0.
     */
    total_supply: MichelsonMap<BigNumber, BigNumber>;
}

export interface VersumInitialStorageLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

export interface VersumInitialStorageOperatorsKey {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumInitialStorageSignedCoCreatorshipKey {
    /** Tezos address. */
    cocreator: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumInitialStorageTokenExtraDataValue {
    /** Simple boolean. */
    contracts_may_hold_token: boolean;

    /**
     * In-memory map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    extra_fields: MichelsonMap<string, string>;

    infusions: VersumInitialStorageTokenExtraDataValueInfusionsItem[];

    /** Nat - arbitrary big integer >= 0. */
    max_per_address: BigNumber;

    /** Tezos address. */
    minter: string;

    /** Date ISO 8601 string. */
    no_transfers_until: { Some: string } | null;

    /** Simple boolean. */
    req_verified_to_own: boolean;

    /** Nat - arbitrary big integer >= 0. */
    royalty: BigNumber;

    splits: VersumInitialStorageTokenExtraDataValueSplitsItem[];
}

export interface VersumInitialStorageTokenExtraDataValueInfusionsItem {
    /** Tezos address. */
    token_address: string;

    token_id_amounts: VersumInitialStorageTokenExtraDataValueInfusionsItemTokenIdAmountsItem[];
}

export interface VersumInitialStorageTokenExtraDataValueInfusionsItemTokenIdAmountsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumInitialStorageTokenExtraDataValueSplitsItem {
    /** Tezos address. */
    address: string;

    /** Nat - arbitrary big integer >= 0. */
    pct: BigNumber;
}

export interface VersumInitialStorageTokenMetadataValue {
    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;

    /**
     * In-memory map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    token_info: MichelsonMap<string, string>;
}

/** Nat - arbitrary big integer >= 0. */
export type Versum24Key = BigNumber;

/** Lambda. */
export type Versum24Value = unknown;

/** Bytes. */
export type VersumExtraDbKey = string;

/** Bytes. */
export type VersumExtraDbValue = string;

export interface VersumLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

/** Nat - arbitrary big integer >= 0. */
export type VersumLedgerValue = BigNumber;

/** Arbitrary string. */
export type VersumMetadataKey = string;

/** Bytes. */
export type VersumMetadataValue = string;

/** Nat - arbitrary big integer >= 0. */
export type VersumMintSlotsKey = BigNumber;

/** Tezos address. */
export type VersumMintSlotsValue = string;

export interface VersumOperatorsKey {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** An empty result. */
export type VersumOperatorsValue = typeof UnitValue;

export interface VersumSignedCoCreatorshipKey {
    /** Tezos address. */
    cocreator: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** An empty result. */
export type VersumSignedCoCreatorshipValue = typeof UnitValue;

/** Nat - arbitrary big integer >= 0. */
export type VersumTokenExtraDataKey = BigNumber;

export interface VersumTokenExtraDataValue {
    /** Simple boolean. */
    contracts_may_hold_token: boolean;

    /**
     * In-memory map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    extra_fields: MichelsonMap<string, string>;

    infusions: VersumTokenExtraDataValueInfusionsItem[];

    /** Nat - arbitrary big integer >= 0. */
    max_per_address: BigNumber;

    /** Tezos address. */
    minter: string;

    /** Date ISO 8601 string. */
    no_transfers_until: { Some: string } | null;

    /** Simple boolean. */
    req_verified_to_own: boolean;

    /** Nat - arbitrary big integer >= 0. */
    royalty: BigNumber;

    splits: VersumTokenExtraDataValueSplitsItem[];
}

export interface VersumTokenExtraDataValueInfusionsItem {
    /** Tezos address. */
    token_address: string;

    token_id_amounts: VersumTokenExtraDataValueInfusionsItemTokenIdAmountsItem[];
}

export interface VersumTokenExtraDataValueInfusionsItemTokenIdAmountsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface VersumTokenExtraDataValueSplitsItem {
    /** Tezos address. */
    address: string;

    /** Nat - arbitrary big integer >= 0. */
    pct: BigNumber;
}

/** Nat - arbitrary big integer >= 0. */
export type VersumTokenMetadataKey = BigNumber;

export interface VersumTokenMetadataValue {
    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;

    /**
     * In-memory map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    token_info: MichelsonMap<string, string>;
}

/** Nat - arbitrary big integer >= 0. */
export type VersumTotalSupplyKey = BigNumber;

/** Nat - arbitrary big integer >= 0. */
export type VersumTotalSupplyValue = BigNumber;
