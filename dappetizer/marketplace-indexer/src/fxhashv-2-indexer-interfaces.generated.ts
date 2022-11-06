/* istanbul ignore next */
/* eslint-disable */

// This file was generated.
// It should NOT be modified manually rather it should be regenerated.
// Contract: KT1U6EHmNxJTkvaWJ4ThczG4FSDaHC21ssvi
// Tezos network: mainnet

import { UnitValue } from '@taquito/michelson-encoder';
import { BigMapAbstraction, MichelsonMap } from '@taquito/taquito';
import { BigNumber } from 'bignumber.js';

export type Fxhashv2Parameter =
    | { entrypoint: 'assign_metadata'; value: Fxhashv2AssignMetadataParameter }
    | { entrypoint: 'balance_of'; value: Fxhashv2BalanceOfParameter }
    | { entrypoint: 'mint'; value: Fxhashv2MintParameter }
    | { entrypoint: 'set_administrator'; value: Fxhashv2SetAdministratorParameter }
    | { entrypoint: 'set_issuer'; value: Fxhashv2SetIssuerParameter }
    | { entrypoint: 'set_metadata'; value: Fxhashv2SetMetadataParameter }
    | { entrypoint: 'set_pause'; value: Fxhashv2SetPauseParameter }
    | { entrypoint: 'set_signer'; value: Fxhashv2SetSignerParameter }
    | { entrypoint: 'set_treasury_address'; value: Fxhashv2SetTreasuryAddressParameter }
    | { entrypoint: 'transfer'; value: Fxhashv2TransferParameter }
    | { entrypoint: 'transfer_xtz_treasury'; value: Fxhashv2TransferXtzTreasuryParameter }
    | { entrypoint: 'update_operators'; value: Fxhashv2UpdateOperatorsParameter };

export type Fxhashv2AssignMetadataParameter = Fxhashv2AssignMetadataParameterItem[];

export interface Fxhashv2AssignMetadataParameterItem {
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

export interface Fxhashv2BalanceOfParameter {
    /** Contract address. */
    callback: string;

    requests: Fxhashv2BalanceOfParameterRequestsItem[];
}

export interface Fxhashv2BalanceOfParameterRequestsItem {
    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface Fxhashv2MintParameter {
    /** Tezos address. */
    address: string;

    /** Nat - arbitrary big integer >= 0. */
    issuer_id: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    iteration: BigNumber;

    /**
     * In-memory map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    metadata: MichelsonMap<string, string>;

    /** Nat - arbitrary big integer >= 0. */
    royalties: BigNumber;

    royalties_split: Fxhashv2MintParameterRoyaltiesSplitItem[];

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface Fxhashv2MintParameterRoyaltiesSplitItem {
    /** Tezos address. */
    address: string;

    /** Nat - arbitrary big integer >= 0. */
    pct: BigNumber;
}

/** Tezos address. */
export type Fxhashv2SetAdministratorParameter = string;

/** Tezos address. */
export type Fxhashv2SetIssuerParameter = string;

/**
 * Big map initial values.
 * 
 * Key of `string`: Arbitrary string.
 * 
 * Value of `string`: Bytes.
 */
export type Fxhashv2SetMetadataParameter = MichelsonMap<string, string>;

/** Simple boolean. */
export type Fxhashv2SetPauseParameter = boolean;

/** Tezos address. */
export type Fxhashv2SetSignerParameter = string;

/** Tezos address. */
export type Fxhashv2SetTreasuryAddressParameter = string;

export type Fxhashv2TransferParameter = Fxhashv2TransferParameterItem[];

export interface Fxhashv2TransferParameterItem {
    /** Tezos address. */
    from_: string;

    txs: Fxhashv2TransferParameterItemTxsItem[];
}

export interface Fxhashv2TransferParameterItemTxsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Tezos address. */
    to_: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** Mutez - arbitrary big integer >= 0. */
export type Fxhashv2TransferXtzTreasuryParameter = BigNumber;

export type Fxhashv2UpdateOperatorsParameter = Fxhashv2UpdateOperatorsParameterItem[];

export interface Fxhashv2UpdateOperatorsParameterItem {
    add_operator?: Fxhashv2UpdateOperatorsParameterItemAddOperator;

    remove_operator?: Fxhashv2UpdateOperatorsParameterItemRemoveOperator;
}

export interface Fxhashv2UpdateOperatorsParameterItemAddOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface Fxhashv2UpdateOperatorsParameterItemRemoveOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface Fxhashv2CurrentStorage {
    /** Tezos address. */
    administrator: string;

    /** Nat - arbitrary big integer >= 0. */
    all_tokens: BigNumber;

    /** Tezos address. */
    issuer: string;

    /**
     * Big map.
     * 
     * Key of `Fxhashv2CurrentStorageLedgerKey`.
     * 
     * Value of `BigNumber`: Nat - arbitrary big integer >= 0.
     */
    ledger: BigMapAbstraction;

    /**
     * Big map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    metadata: BigMapAbstraction;

    /**
     * Big map.
     * 
     * Key of `Fxhashv2CurrentStorageOperatorsKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    operators: BigMapAbstraction;

    /** Simple boolean. */
    paused: boolean;

    /** Tezos address. */
    signer: string;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `Fxhashv2CurrentStorageTokenDataValue`.
     */
    token_data: BigMapAbstraction;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `Fxhashv2CurrentStorageTokenMetadataValue`.
     */
    token_metadata: BigMapAbstraction;

    /** Tezos address. */
    treasury_address: string;
}

export interface Fxhashv2CurrentStorageLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

export interface Fxhashv2CurrentStorageOperatorsKey {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface Fxhashv2CurrentStorageTokenDataValue {
    /** Simple boolean. */
    assigned: boolean;

    /** Nat - arbitrary big integer >= 0. */
    issuer_id: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    iteration: BigNumber;

    /** Tezos address. */
    minter: string;

    /** Nat - arbitrary big integer >= 0. */
    royalties: BigNumber;

    royalties_split: Fxhashv2CurrentStorageTokenDataValueRoyaltiesSplitItem[];
}

export interface Fxhashv2CurrentStorageTokenDataValueRoyaltiesSplitItem {
    /** Tezos address. */
    address: string;

    /** Nat - arbitrary big integer >= 0. */
    pct: BigNumber;
}

export interface Fxhashv2CurrentStorageTokenMetadataValue {
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

export interface Fxhashv2ChangedStorage {
    /** Tezos address. */
    administrator: string;

    /** Nat - arbitrary big integer >= 0. */
    all_tokens: BigNumber;

    /** Tezos address. */
    issuer: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    ledger: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    metadata: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    operators: string;

    /** Simple boolean. */
    paused: boolean;

    /** Tezos address. */
    signer: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    token_data: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    token_metadata: string;

    /** Tezos address. */
    treasury_address: string;
}

export interface Fxhashv2InitialStorage {
    /** Tezos address. */
    administrator: string;

    /** Nat - arbitrary big integer >= 0. */
    all_tokens: BigNumber;

    /** Tezos address. */
    issuer: string;

    /**
     * Big map initial values.
     * 
     * Key of `Fxhashv2InitialStorageLedgerKey`.
     * 
     * Value of `BigNumber`: Nat - arbitrary big integer >= 0.
     */
    ledger: MichelsonMap<Fxhashv2InitialStorageLedgerKey, BigNumber>;

    /**
     * Big map initial values.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    metadata: MichelsonMap<string, string>;

    /**
     * Big map initial values.
     * 
     * Key of `Fxhashv2InitialStorageOperatorsKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    operators: MichelsonMap<Fxhashv2InitialStorageOperatorsKey, typeof UnitValue>;

    /** Simple boolean. */
    paused: boolean;

    /** Tezos address. */
    signer: string;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `Fxhashv2InitialStorageTokenDataValue`.
     */
    token_data: MichelsonMap<BigNumber, Fxhashv2InitialStorageTokenDataValue>;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `Fxhashv2InitialStorageTokenMetadataValue`.
     */
    token_metadata: MichelsonMap<BigNumber, Fxhashv2InitialStorageTokenMetadataValue>;

    /** Tezos address. */
    treasury_address: string;
}

export interface Fxhashv2InitialStorageLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

export interface Fxhashv2InitialStorageOperatorsKey {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface Fxhashv2InitialStorageTokenDataValue {
    /** Simple boolean. */
    assigned: boolean;

    /** Nat - arbitrary big integer >= 0. */
    issuer_id: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    iteration: BigNumber;

    /** Tezos address. */
    minter: string;

    /** Nat - arbitrary big integer >= 0. */
    royalties: BigNumber;

    royalties_split: Fxhashv2InitialStorageTokenDataValueRoyaltiesSplitItem[];
}

export interface Fxhashv2InitialStorageTokenDataValueRoyaltiesSplitItem {
    /** Tezos address. */
    address: string;

    /** Nat - arbitrary big integer >= 0. */
    pct: BigNumber;
}

export interface Fxhashv2InitialStorageTokenMetadataValue {
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

export interface Fxhashv2LedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

/** Nat - arbitrary big integer >= 0. */
export type Fxhashv2LedgerValue = BigNumber;

/** Arbitrary string. */
export type Fxhashv2MetadataKey = string;

/** Bytes. */
export type Fxhashv2MetadataValue = string;

export interface Fxhashv2OperatorsKey {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** An empty result. */
export type Fxhashv2OperatorsValue = typeof UnitValue;

/** Nat - arbitrary big integer >= 0. */
export type Fxhashv2TokenDataKey = BigNumber;

export interface Fxhashv2TokenDataValue {
    /** Simple boolean. */
    assigned: boolean;

    /** Nat - arbitrary big integer >= 0. */
    issuer_id: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    iteration: BigNumber;

    /** Tezos address. */
    minter: string;

    /** Nat - arbitrary big integer >= 0. */
    royalties: BigNumber;

    royalties_split: Fxhashv2TokenDataValueRoyaltiesSplitItem[];
}

export interface Fxhashv2TokenDataValueRoyaltiesSplitItem {
    /** Tezos address. */
    address: string;

    /** Nat - arbitrary big integer >= 0. */
    pct: BigNumber;
}

/** Nat - arbitrary big integer >= 0. */
export type Fxhashv2TokenMetadataKey = BigNumber;

export interface Fxhashv2TokenMetadataValue {
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
