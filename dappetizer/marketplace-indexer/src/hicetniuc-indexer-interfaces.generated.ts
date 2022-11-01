/* istanbul ignore next */
/* eslint-disable */

// This file was generated.
// It should NOT be modified manually rather it should be regenerated.
// Contract: KT1RJ6PbjHpwc3M5rw5s2Nbmefwbuwbdxton
// Tezos network: mainnet

import { UnitValue } from '@taquito/michelson-encoder';
import { BigMapAbstraction, MichelsonMap } from '@taquito/taquito';
import { BigNumber } from 'bignumber.js';

export type HicetniucParameter =
    | { entrypoint: 'balance_of'; value: HicetniucBalanceOfParameter }
    | { entrypoint: 'hDAO_batch'; value: HicetniucHDaoBatchParameter }
    | { entrypoint: 'mint'; value: HicetniucMintParameter }
    | { entrypoint: 'set_administrator'; value: HicetniucSetAdministratorParameter }
    | { entrypoint: 'set_pause'; value: HicetniucSetPauseParameter }
    | { entrypoint: 'token_metadata'; value: HicetniucTokenMetadataParameter }
    | { entrypoint: 'transfer'; value: HicetniucTransferParameter }
    | { entrypoint: 'update_operators'; value: HicetniucUpdateOperatorsParameter };

export interface HicetniucBalanceOfParameter {
    /** Contract address. */
    callback: string;

    requests: HicetniucBalanceOfParameterRequestsItem[];
}

export interface HicetniucBalanceOfParameterRequestsItem {
    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export type HicetniucHDaoBatchParameter = HicetniucHDaoBatchParameterItem[];

export interface HicetniucHDaoBatchParameterItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Tezos address. */
    to_: string;
}

export interface HicetniucMintParameter {
    /** Tezos address. */
    address: string;

    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

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

/** Tezos address. */
export type HicetniucSetAdministratorParameter = string;

/** Simple boolean. */
export type HicetniucSetPauseParameter = boolean;

export interface HicetniucTokenMetadataParameter {
    /** Lambda. */
    handler: unknown;

    /** Array of: Nat - arbitrary big integer >= 0. */
    token_ids: BigNumber[];
}

export type HicetniucTransferParameter = HicetniucTransferParameterItem[];

export interface HicetniucTransferParameterItem {
    /** Tezos address. */
    from_: string;

    txs: HicetniucTransferParameterItemTxsItem[];
}

export interface HicetniucTransferParameterItemTxsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Tezos address. */
    to_: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export type HicetniucUpdateOperatorsParameter = HicetniucUpdateOperatorsParameterItem[];

export interface HicetniucUpdateOperatorsParameterItem {
    add_operator?: HicetniucUpdateOperatorsParameterItemAddOperator;

    remove_operator?: HicetniucUpdateOperatorsParameterItemRemoveOperator;
}

export interface HicetniucUpdateOperatorsParameterItemAddOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface HicetniucUpdateOperatorsParameterItemRemoveOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface HicetniucCurrentStorage {
    /** Tezos address. */
    administrator: string;

    /** Nat - arbitrary big integer >= 0. */
    all_tokens: BigNumber;

    /**
     * Big map.
     * 
     * Key of `HicetniucCurrentStorageLedgerKey`.
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
     * Key of `HicetniucCurrentStorageOperatorsKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    operators: BigMapAbstraction;

    /** Simple boolean. */
    paused: boolean;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `HicetniucCurrentStorageTokenMetadataValue`.
     */
    token_metadata: BigMapAbstraction;
}

export interface HicetniucCurrentStorageLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

export interface HicetniucCurrentStorageOperatorsKey {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface HicetniucCurrentStorageTokenMetadataValue {
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

export interface HicetniucChangedStorage {
    /** Tezos address. */
    administrator: string;

    /** Nat - arbitrary big integer >= 0. */
    all_tokens: BigNumber;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    ledger: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    metadata: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    operators: string;

    /** Simple boolean. */
    paused: boolean;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    token_metadata: string;
}

export interface HicetniucInitialStorage {
    /** Tezos address. */
    administrator: string;

    /** Nat - arbitrary big integer >= 0. */
    all_tokens: BigNumber;

    /**
     * Big map initial values.
     * 
     * Key of `HicetniucInitialStorageLedgerKey`.
     * 
     * Value of `BigNumber`: Nat - arbitrary big integer >= 0.
     */
    ledger: MichelsonMap<HicetniucInitialStorageLedgerKey, BigNumber>;

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
     * Key of `HicetniucInitialStorageOperatorsKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    operators: MichelsonMap<HicetniucInitialStorageOperatorsKey, typeof UnitValue>;

    /** Simple boolean. */
    paused: boolean;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `HicetniucInitialStorageTokenMetadataValue`.
     */
    token_metadata: MichelsonMap<BigNumber, HicetniucInitialStorageTokenMetadataValue>;
}

export interface HicetniucInitialStorageLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

export interface HicetniucInitialStorageOperatorsKey {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface HicetniucInitialStorageTokenMetadataValue {
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

export interface HicetniucLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

/** Nat - arbitrary big integer >= 0. */
export type HicetniucLedgerValue = BigNumber;

/** Arbitrary string. */
export type HicetniucMetadataKey = string;

/** Bytes. */
export type HicetniucMetadataValue = string;

export interface HicetniucOperatorsKey {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** An empty result. */
export type HicetniucOperatorsValue = typeof UnitValue;

/** Nat - arbitrary big integer >= 0. */
export type HicetniucTokenMetadataKey = BigNumber;

export interface HicetniucTokenMetadataValue {
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
