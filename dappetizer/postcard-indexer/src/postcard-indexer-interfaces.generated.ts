/* istanbul ignore next */
/* eslint-disable */

// This file was generated.
// It should NOT be modified manually rather it should be regenerated.
// Contract: KT1ESGez4dEuDjjNt4k2HPAK5Nzh7e8X8jyX
// Tezos network: ghostnet

import { UnitValue } from '@taquito/michelson-encoder';
import { BigMapAbstraction, MichelsonMap } from '@taquito/taquito';
import { BigNumber } from 'bignumber.js';

export type PostcardParameter =
    | { entrypoint: 'add_trustee'; value: PostcardAddTrusteeParameter }
    | { entrypoint: 'admin'; value: PostcardAdminParameter }
    | { entrypoint: 'assets'; value: PostcardAssetsParameter }
    | { entrypoint: 'balance_of'; value: PostcardBalanceOfParameter }
    | { entrypoint: 'confirm_admin'; value: PostcardConfirmAdminParameter }
    | { entrypoint: 'mail_postcard'; value: PostcardMailPostcardParameter }
    | { entrypoint: 'mint_postcard'; value: PostcardMintPostcardParameter }
    | { entrypoint: 'pause'; value: PostcardPauseParameter }
    | { entrypoint: 'postcards'; value: PostcardPostcardsParameter }
    | { entrypoint: 'remove_trustee'; value: PostcardRemoveTrusteeParameter }
    | { entrypoint: 'set_admin'; value: PostcardSetAdminParameter }
    | { entrypoint: 'stamp_postcard'; value: PostcardStampPostcardParameter }
    | { entrypoint: 'transfer'; value: PostcardTransferParameter }
    | { entrypoint: 'trustee'; value: PostcardTrusteeParameter }
    | { entrypoint: 'update_operators'; value: PostcardUpdateOperatorsParameter };

/** Tezos address. */
export type PostcardAddTrusteeParameter = string;

export interface PostcardAdminParameter {
    /** An empty result. */
    confirm_admin?: typeof UnitValue;

    /** Simple boolean. */
    pause?: boolean;

    /** Tezos address. */
    set_admin?: string;
}

export interface PostcardAssetsParameter {
    balance_of?: PostcardAssetsParameterBalanceOf;

    transfer?: PostcardAssetsParameterTransferItem[];

    update_operators?: PostcardAssetsParameterUpdateOperatorsItem[];
}

export interface PostcardAssetsParameterBalanceOf {
    /** Contract address. */
    callback: string;

    requests: PostcardAssetsParameterBalanceOfRequestsItem[];
}

export interface PostcardAssetsParameterBalanceOfRequestsItem {
    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface PostcardAssetsParameterTransferItem {
    /** Tezos address. */
    from_: string;

    txs: PostcardAssetsParameterTransferItemTxsItem[];
}

export interface PostcardAssetsParameterTransferItemTxsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Tezos address. */
    to_: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface PostcardAssetsParameterUpdateOperatorsItem {
    add_operator?: PostcardAssetsParameterUpdateOperatorsItemAddOperator;

    remove_operator?: PostcardAssetsParameterUpdateOperatorsItemRemoveOperator;
}

export interface PostcardAssetsParameterUpdateOperatorsItemAddOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface PostcardAssetsParameterUpdateOperatorsItemRemoveOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface PostcardBalanceOfParameter {
    /** Contract address. */
    callback: string;

    requests: PostcardBalanceOfParameterRequestsItem[];
}

export interface PostcardBalanceOfParameterRequestsItem {
    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** An empty result. */
export type PostcardConfirmAdminParameter = typeof UnitValue;

export type PostcardMailPostcardParameter = PostcardMailPostcardParameterItem[];

export interface PostcardMailPostcardParameterItem {
    /** Tezos address. */
    address: string;

    /** Nat - arbitrary big integer >= 0. */
    counter: BigNumber;

    /** Key. */
    pk: string;

    /** Tezos address. */
    postman: string;

    /** Signature. */
    sig: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;

    /** Bytes. */
    token_info_uri: string;
}

export type PostcardMintPostcardParameter = PostcardMintPostcardParameterItem[];

export interface PostcardMintPostcardParameterItem {
    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;

    /** Bytes. */
    token_info_uri: string;
}

/** Simple boolean. */
export type PostcardPauseParameter = boolean;

export interface PostcardPostcardsParameter {
    mail_postcard?: PostcardPostcardsParameterMailPostcardItem[];

    mint_postcard?: PostcardPostcardsParameterMintPostcardItem[];

    stamp_postcard?: PostcardPostcardsParameterStampPostcardItem[];
}

export interface PostcardPostcardsParameterMailPostcardItem {
    /** Tezos address. */
    address: string;

    /** Nat - arbitrary big integer >= 0. */
    counter: BigNumber;

    /** Key. */
    pk: string;

    /** Tezos address. */
    postman: string;

    /** Signature. */
    sig: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;

    /** Bytes. */
    token_info_uri: string;
}

export interface PostcardPostcardsParameterMintPostcardItem {
    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;

    /** Bytes. */
    token_info_uri: string;
}

export interface PostcardPostcardsParameterStampPostcardItem {
    /** Nat - arbitrary big integer >= 0. */
    counter: BigNumber;

    /** Key. */
    pk: string;

    /** Tezos address. */
    postman: string;

    /** Signature. */
    sig: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;

    /** Bytes. */
    token_info_uri: string;
}

/** Tezos address. */
export type PostcardRemoveTrusteeParameter = string;

/** Tezos address. */
export type PostcardSetAdminParameter = string;

export type PostcardStampPostcardParameter = PostcardStampPostcardParameterItem[];

export interface PostcardStampPostcardParameterItem {
    /** Nat - arbitrary big integer >= 0. */
    counter: BigNumber;

    /** Key. */
    pk: string;

    /** Tezos address. */
    postman: string;

    /** Signature. */
    sig: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;

    /** Bytes. */
    token_info_uri: string;
}

export type PostcardTransferParameter = PostcardTransferParameterItem[];

export interface PostcardTransferParameterItem {
    /** Tezos address. */
    from_: string;

    txs: PostcardTransferParameterItemTxsItem[];
}

export interface PostcardTransferParameterItemTxsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Tezos address. */
    to_: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface PostcardTrusteeParameter {
    /** Tezos address. */
    add_trustee?: string;

    /** Tezos address. */
    remove_trustee?: string;
}

export type PostcardUpdateOperatorsParameter = PostcardUpdateOperatorsParameterItem[];

export interface PostcardUpdateOperatorsParameterItem {
    add_operator?: PostcardUpdateOperatorsParameterItemAddOperator;

    remove_operator?: PostcardUpdateOperatorsParameterItemRemoveOperator;
}

export interface PostcardUpdateOperatorsParameterItemAddOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface PostcardUpdateOperatorsParameterItemRemoveOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface PostcardCurrentStorage {
    admin: PostcardCurrentStorageAdmin;

    assets: PostcardCurrentStorageAssets;

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
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `PostcardCurrentStoragePostcardsValue`.
     */
    postcards: BigMapAbstraction;

    trustee: PostcardCurrentStorageTrustee;
}

export interface PostcardCurrentStorageAdmin {
    /** Tezos address. */
    admin: string;

    /** Simple boolean. */
    paused: boolean;

    /** Tezos address. */
    pending_admin: string | null;
}

export interface PostcardCurrentStorageAssets {
    /**
     * Big map.
     * 
     * Key of `PostcardCurrentStorageAssetsLedgerKey`.
     * 
     * Value of `BigNumber`: Nat - arbitrary big integer >= 0.
     */
    ledger: BigMapAbstraction;

    /**
     * Big map.
     * 
     * Key of `PostcardCurrentStorageAssetsOperatorsKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    operators: BigMapAbstraction;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `PostcardCurrentStorageAssetsTokenMetadataValue`.
     */
    token_metadata: BigMapAbstraction;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `BigNumber`: Nat - arbitrary big integer >= 0.
     */
    token_total_supply: BigMapAbstraction;
}

export interface PostcardCurrentStorageAssetsLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

export interface PostcardCurrentStorageAssetsOperatorsKey {
    /** Tezos address. */
    '0': string;

    /** Tezos address. */
    '1': string;

    /** Nat - arbitrary big integer >= 0. */
    '2': BigNumber;
}

export interface PostcardCurrentStorageAssetsTokenMetadataValue {
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

export interface PostcardCurrentStoragePostcardsValue {
    /** Nat - arbitrary big integer >= 0. */
    counter: BigNumber;

    /** Tezos address. */
    postman: string;

    /** Simple boolean. */
    stamped: boolean;
}

export interface PostcardCurrentStorageTrustee {
    /** Nat - arbitrary big integer >= 0. */
    max_trustee: BigNumber;

    /** Array of: Tezos address. */
    trustees: string[];
}

export interface PostcardChangedStorage {
    admin: PostcardChangedStorageAdmin;

    assets: PostcardChangedStorageAssets;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    metadata: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    postcards: string;

    trustee: PostcardChangedStorageTrustee;
}

export interface PostcardChangedStorageAdmin {
    /** Tezos address. */
    admin: string;

    /** Simple boolean. */
    paused: boolean;

    /** Tezos address. */
    pending_admin: string | null;
}

export interface PostcardChangedStorageAssets {
    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    ledger: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    operators: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    token_metadata: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    token_total_supply: string;
}

export interface PostcardChangedStorageTrustee {
    /** Nat - arbitrary big integer >= 0. */
    max_trustee: BigNumber;

    /** Array of: Tezos address. */
    trustees: string[];
}

export interface PostcardInitialStorage {
    admin: PostcardInitialStorageAdmin;

    assets: PostcardInitialStorageAssets;

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
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `PostcardInitialStoragePostcardsValue`.
     */
    postcards: MichelsonMap<BigNumber, PostcardInitialStoragePostcardsValue>;

    trustee: PostcardInitialStorageTrustee;
}

export interface PostcardInitialStorageAdmin {
    /** Tezos address. */
    admin: string;

    /** Simple boolean. */
    paused: boolean;

    /** Tezos address. */
    pending_admin: string | null;
}

export interface PostcardInitialStorageAssets {
    /**
     * Big map initial values.
     * 
     * Key of `PostcardInitialStorageAssetsLedgerKey`.
     * 
     * Value of `BigNumber`: Nat - arbitrary big integer >= 0.
     */
    ledger: MichelsonMap<PostcardInitialStorageAssetsLedgerKey, BigNumber>;

    /**
     * Big map initial values.
     * 
     * Key of `PostcardInitialStorageAssetsOperatorsKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    operators: MichelsonMap<PostcardInitialStorageAssetsOperatorsKey, typeof UnitValue>;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `PostcardInitialStorageAssetsTokenMetadataValue`.
     */
    token_metadata: MichelsonMap<BigNumber, PostcardInitialStorageAssetsTokenMetadataValue>;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `BigNumber`: Nat - arbitrary big integer >= 0.
     */
    token_total_supply: MichelsonMap<BigNumber, BigNumber>;
}

export interface PostcardInitialStorageAssetsLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

export interface PostcardInitialStorageAssetsOperatorsKey {
    /** Tezos address. */
    '0': string;

    /** Tezos address. */
    '1': string;

    /** Nat - arbitrary big integer >= 0. */
    '2': BigNumber;
}

export interface PostcardInitialStorageAssetsTokenMetadataValue {
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

export interface PostcardInitialStoragePostcardsValue {
    /** Nat - arbitrary big integer >= 0. */
    counter: BigNumber;

    /** Tezos address. */
    postman: string;

    /** Simple boolean. */
    stamped: boolean;
}

export interface PostcardInitialStorageTrustee {
    /** Nat - arbitrary big integer >= 0. */
    max_trustee: BigNumber;

    /** Array of: Tezos address. */
    trustees: string[];
}

export interface PostcardAssetsLedgerKey {
    /** Tezos address. */
    '0': string;

    /** Nat - arbitrary big integer >= 0. */
    '1': BigNumber;
}

/** Nat - arbitrary big integer >= 0. */
export type PostcardAssetsLedgerValue = BigNumber;

export interface PostcardAssetsOperatorsKey {
    /** Tezos address. */
    '0': string;

    /** Tezos address. */
    '1': string;

    /** Nat - arbitrary big integer >= 0. */
    '2': BigNumber;
}

/** An empty result. */
export type PostcardAssetsOperatorsValue = typeof UnitValue;

/** Nat - arbitrary big integer >= 0. */
export type PostcardAssetsTokenMetadataKey = BigNumber;

export interface PostcardAssetsTokenMetadataValue {
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
export type PostcardAssetsTokenTotalSupplyKey = BigNumber;

/** Nat - arbitrary big integer >= 0. */
export type PostcardAssetsTokenTotalSupplyValue = BigNumber;

/** Arbitrary string. */
export type PostcardMetadataKey = string;

/** Bytes. */
export type PostcardMetadataValue = string;

/** Nat - arbitrary big integer >= 0. */
export type PostcardPostcardsKey = BigNumber;

export interface PostcardPostcardsValue {
    /** Nat - arbitrary big integer >= 0. */
    counter: BigNumber;

    /** Tezos address. */
    postman: string;

    /** Simple boolean. */
    stamped: boolean;
}
