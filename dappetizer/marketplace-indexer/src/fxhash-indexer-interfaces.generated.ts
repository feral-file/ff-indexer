/* istanbul ignore next */
/* eslint-disable */

// This file was generated.
// It should NOT be modified manually rather it should be regenerated.
// Contract: KT1KEa8z6vWXDJrVqtMrAeDVzsvxat3kHaCE
// Tezos network: mainnet

import { UnitValue } from '@taquito/michelson-encoder';
import { BigMapAbstraction, MichelsonMap } from '@taquito/taquito';
import { BigNumber } from 'bignumber.js';

export type FxhashParameter =
    | { entrypoint: 'assign_metadata'; value: FxhashAssignMetadataParameter }
    | { entrypoint: 'balance_of'; value: FxhashBalanceOfParameter }
    | { entrypoint: 'mint'; value: FxhashMintParameter }
    | { entrypoint: 'set_administrator'; value: FxhashSetAdministratorParameter }
    | { entrypoint: 'set_issuer'; value: FxhashSetIssuerParameter }
    | { entrypoint: 'set_metadata'; value: FxhashSetMetadataParameter }
    | { entrypoint: 'set_pause'; value: FxhashSetPauseParameter }
    | { entrypoint: 'set_signer'; value: FxhashSetSignerParameter }
    | { entrypoint: 'set_treasury_address'; value: FxhashSetTreasuryAddressParameter }
    | { entrypoint: 'transfer'; value: FxhashTransferParameter }
    | { entrypoint: 'transfer_xtz_treasury'; value: FxhashTransferXtzTreasuryParameter }
    | { entrypoint: 'update_operators'; value: FxhashUpdateOperatorsParameter };

export interface FxhashAssignMetadataParameter {
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

export interface FxhashBalanceOfParameter {
    /** Contract address. */
    callback: string;

    requests: FxhashBalanceOfParameterRequestsItem[];
}

export interface FxhashBalanceOfParameterRequestsItem {
    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface FxhashMintParameter {
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

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** Tezos address. */
export type FxhashSetAdministratorParameter = string;

/** Tezos address. */
export type FxhashSetIssuerParameter = string;

export interface FxhashSetMetadataParameter {
    /** Arbitrary string. */
    k: string;

    /** Bytes. */
    v: string;
}

/** Simple boolean. */
export type FxhashSetPauseParameter = boolean;

/** Tezos address. */
export type FxhashSetSignerParameter = string;

/** Tezos address. */
export type FxhashSetTreasuryAddressParameter = string;

export type FxhashTransferParameter = FxhashTransferParameterItem[];

export interface FxhashTransferParameterItem {
    /** Tezos address. */
    from_: string;

    txs: FxhashTransferParameterItemTxsItem[];
}

export interface FxhashTransferParameterItemTxsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Tezos address. */
    to_: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** Mutez - arbitrary big integer >= 0. */
export type FxhashTransferXtzTreasuryParameter = BigNumber;

export type FxhashUpdateOperatorsParameter = FxhashUpdateOperatorsParameterItem[];

export interface FxhashUpdateOperatorsParameterItem {
    add_operator?: FxhashUpdateOperatorsParameterItemAddOperator;

    remove_operator?: FxhashUpdateOperatorsParameterItemRemoveOperator;
}

export interface FxhashUpdateOperatorsParameterItemAddOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface FxhashUpdateOperatorsParameterItemRemoveOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface FxhashCurrentStorage {
    /** Tezos address. */
    administrator: string;

    /** Nat - arbitrary big integer >= 0. */
    all_tokens: BigNumber;

    /** Tezos address. */
    issuer: string;

    /**
     * Big map.
     * 
     * Key of `FxhashCurrentStorageLedgerKey`.
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
     * Key of `FxhashCurrentStorageOperatorsKey`.
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
     * Value of `FxhashCurrentStorageTokenDataValue`.
     */
    token_data: BigMapAbstraction;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `FxhashCurrentStorageTokenMetadataValue`.
     */
    token_metadata: BigMapAbstraction;

    /** Tezos address. */
    treasury_address: string;
}

export interface FxhashCurrentStorageLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

export interface FxhashCurrentStorageOperatorsKey {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface FxhashCurrentStorageTokenDataValue {
    /** Simple boolean. */
    assigned: boolean;

    /** Nat - arbitrary big integer >= 0. */
    issuer_id: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    iteration: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    royalties: BigNumber;
}

export interface FxhashCurrentStorageTokenMetadataValue {
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

export interface FxhashChangedStorage {
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

export interface FxhashInitialStorage {
    /** Tezos address. */
    administrator: string;

    /** Nat - arbitrary big integer >= 0. */
    all_tokens: BigNumber;

    /** Tezos address. */
    issuer: string;

    /**
     * Big map initial values.
     * 
     * Key of `FxhashInitialStorageLedgerKey`.
     * 
     * Value of `BigNumber`: Nat - arbitrary big integer >= 0.
     */
    ledger: MichelsonMap<FxhashInitialStorageLedgerKey, BigNumber>;

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
     * Key of `FxhashInitialStorageOperatorsKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    operators: MichelsonMap<FxhashInitialStorageOperatorsKey, typeof UnitValue>;

    /** Simple boolean. */
    paused: boolean;

    /** Tezos address. */
    signer: string;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `FxhashInitialStorageTokenDataValue`.
     */
    token_data: MichelsonMap<BigNumber, FxhashInitialStorageTokenDataValue>;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `FxhashInitialStorageTokenMetadataValue`.
     */
    token_metadata: MichelsonMap<BigNumber, FxhashInitialStorageTokenMetadataValue>;

    /** Tezos address. */
    treasury_address: string;
}

export interface FxhashInitialStorageLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

export interface FxhashInitialStorageOperatorsKey {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface FxhashInitialStorageTokenDataValue {
    /** Simple boolean. */
    assigned: boolean;

    /** Nat - arbitrary big integer >= 0. */
    issuer_id: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    iteration: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    royalties: BigNumber;
}

export interface FxhashInitialStorageTokenMetadataValue {
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

export interface FxhashLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

/** Nat - arbitrary big integer >= 0. */
export type FxhashLedgerValue = BigNumber;

/** Arbitrary string. */
export type FxhashMetadataKey = string;

/** Bytes. */
export type FxhashMetadataValue = string;

export interface FxhashOperatorsKey {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** An empty result. */
export type FxhashOperatorsValue = typeof UnitValue;

/** Nat - arbitrary big integer >= 0. */
export type FxhashTokenDataKey = BigNumber;

export interface FxhashTokenDataValue {
    /** Simple boolean. */
    assigned: boolean;

    /** Nat - arbitrary big integer >= 0. */
    issuer_id: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    iteration: BigNumber;

    /** Nat - arbitrary big integer >= 0. */
    royalties: BigNumber;
}

/** Nat - arbitrary big integer >= 0. */
export type FxhashTokenMetadataKey = BigNumber;

export interface FxhashTokenMetadataValue {
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
