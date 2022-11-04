/* istanbul ignore next */
/* eslint-disable */

// This file was generated.
// It should NOT be modified manually rather it should be regenerated.
// Contract: KT1RnhKKsAD7ScFi3Nb7HKK2hnPCqXcbNG3k
// Tezos network: mainnet

import { UnitValue } from '@taquito/michelson-encoder';
import { BigMapAbstraction, MichelsonMap } from '@taquito/taquito';
import { BigNumber } from 'bignumber.js';

export type FeralFileV1Parameter =
    | { entrypoint: 'add_trustee'; value: FeralFileV1AddTrusteeParameter }
    | { entrypoint: 'admin'; value: FeralFileV1AdminParameter }
    | { entrypoint: 'assets'; value: FeralFileV1AssetsParameter }
    | { entrypoint: 'authorized_transfer'; value: FeralFileV1AuthorizedTransferParameter }
    | { entrypoint: 'balance_of'; value: FeralFileV1BalanceOfParameter }
    | { entrypoint: 'burn_editions'; value: FeralFileV1BurnEditionsParameter }
    | { entrypoint: 'confirm_admin'; value: FeralFileV1ConfirmAdminParameter }
    | { entrypoint: 'mint_editions'; value: FeralFileV1MintEditionsParameter }
    | { entrypoint: 'minter'; value: FeralFileV1MinterParameter }
    | { entrypoint: 'register_artworks'; value: FeralFileV1RegisterArtworksParameter }
    | { entrypoint: 'remove_trustee'; value: FeralFileV1RemoveTrusteeParameter }
    | { entrypoint: 'set_admin'; value: FeralFileV1SetAdminParameter }
    | { entrypoint: 'transfer'; value: FeralFileV1TransferParameter }
    | { entrypoint: 'trustee'; value: FeralFileV1TrusteeParameter }
    | { entrypoint: 'update_edition_metadata'; value: FeralFileV1UpdateEditionMetadataParameter }
    | { entrypoint: 'update_operators'; value: FeralFileV1UpdateOperatorsParameter };

/** Tezos address. */
export type FeralFileV1AddTrusteeParameter = string;

export interface FeralFileV1AdminParameter {
    /** An empty result. */
    confirm_admin?: typeof UnitValue;

    /** Tezos address. */
    set_admin?: string;
}

export interface FeralFileV1AssetsParameter {
    balance_of?: FeralFileV1AssetsParameterBalanceOf;

    transfer?: FeralFileV1AssetsParameterTransferItem[];

    update_operators?: FeralFileV1AssetsParameterUpdateOperatorsItem[];
}

export interface FeralFileV1AssetsParameterBalanceOf {
    /** Contract address. */
    callback: string;

    requests: FeralFileV1AssetsParameterBalanceOfRequestsItem[];
}

export interface FeralFileV1AssetsParameterBalanceOfRequestsItem {
    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface FeralFileV1AssetsParameterTransferItem {
    /** Tezos address. */
    from_: string;

    txs: FeralFileV1AssetsParameterTransferItemTxsItem[];
}

export interface FeralFileV1AssetsParameterTransferItemTxsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Tezos address. */
    to_: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface FeralFileV1AssetsParameterUpdateOperatorsItem {
    add_operator?: FeralFileV1AssetsParameterUpdateOperatorsItemAddOperator;

    remove_operator?: FeralFileV1AssetsParameterUpdateOperatorsItemRemoveOperator;
}

export interface FeralFileV1AssetsParameterUpdateOperatorsItemAddOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface FeralFileV1AssetsParameterUpdateOperatorsItemRemoveOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export type FeralFileV1AuthorizedTransferParameter = FeralFileV1AuthorizedTransferParameterItem[];

export interface FeralFileV1AuthorizedTransferParameterItem {
    /** Date ISO 8601 string. */
    expiry: string;

    /** Tezos address. */
    from_: string;

    /** Key. */
    pk: string;

    txs: FeralFileV1AuthorizedTransferParameterItemTxsItem[];
}

export interface FeralFileV1AuthorizedTransferParameterItemTxsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Signature. */
    sig: string;

    /** Tezos address. */
    to_: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface FeralFileV1BalanceOfParameter {
    /** Contract address. */
    callback: string;

    requests: FeralFileV1BalanceOfParameterRequestsItem[];
}

export interface FeralFileV1BalanceOfParameterRequestsItem {
    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

/** Array of: Nat - arbitrary big integer >= 0. */
export type FeralFileV1BurnEditionsParameter = BigNumber[];

/** An empty result. */
export type FeralFileV1ConfirmAdminParameter = typeof UnitValue;

export type FeralFileV1MintEditionsParameter = FeralFileV1MintEditionsParameterItem[];

export interface FeralFileV1MintEditionsParameterItem {
    /** Tezos address. */
    owner: string;

    tokens: FeralFileV1MintEditionsParameterItemTokensItem[];
}

export interface FeralFileV1MintEditionsParameterItemTokensItem {
    /** Bytes. */
    artwork_id: string;

    /** Nat - arbitrary big integer >= 0. */
    edition: BigNumber;

    /**
     * In-memory map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    token_info: MichelsonMap<string, string>;
}

export interface FeralFileV1MinterParameter {
    mint_editions?: FeralFileV1MinterParameterMintEditionsItem[];

    register_artworks?: FeralFileV1MinterParameterRegisterArtworksItem[];

    update_edition_metadata?: FeralFileV1MinterParameterUpdateEditionMetadataItem[];
}

export interface FeralFileV1MinterParameterMintEditionsItem {
    /** Tezos address. */
    owner: string;

    tokens: FeralFileV1MinterParameterMintEditionsItemTokensItem[];
}

export interface FeralFileV1MinterParameterMintEditionsItemTokensItem {
    /** Bytes. */
    artwork_id: string;

    /** Nat - arbitrary big integer >= 0. */
    edition: BigNumber;

    /**
     * In-memory map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    token_info: MichelsonMap<string, string>;
}

export interface FeralFileV1MinterParameterRegisterArtworksItem {
    /** Arbitrary string. */
    artist_name: string;

    /** Bytes. */
    fingerprint: string;

    /** Nat - arbitrary big integer >= 0. */
    max_edition: BigNumber;

    /** Tezos address. */
    royalty_address: string;

    /** Arbitrary string. */
    title: string;
}

export interface FeralFileV1MinterParameterUpdateEditionMetadataItem {
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

export type FeralFileV1RegisterArtworksParameter = FeralFileV1RegisterArtworksParameterItem[];

export interface FeralFileV1RegisterArtworksParameterItem {
    /** Arbitrary string. */
    artist_name: string;

    /** Bytes. */
    fingerprint: string;

    /** Nat - arbitrary big integer >= 0. */
    max_edition: BigNumber;

    /** Tezos address. */
    royalty_address: string;

    /** Arbitrary string. */
    title: string;
}

/** Tezos address. */
export type FeralFileV1RemoveTrusteeParameter = string;

/** Tezos address. */
export type FeralFileV1SetAdminParameter = string;

export type FeralFileV1TransferParameter = FeralFileV1TransferParameterItem[];

export interface FeralFileV1TransferParameterItem {
    /** Tezos address. */
    from_: string;

    txs: FeralFileV1TransferParameterItemTxsItem[];
}

export interface FeralFileV1TransferParameterItemTxsItem {
    /** Nat - arbitrary big integer >= 0. */
    amount: BigNumber;

    /** Tezos address. */
    to_: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface FeralFileV1TrusteeParameter {
    /** Tezos address. */
    add_trustee?: string;

    /** Tezos address. */
    remove_trustee?: string;
}

export type FeralFileV1UpdateEditionMetadataParameter = FeralFileV1UpdateEditionMetadataParameterItem[];

export interface FeralFileV1UpdateEditionMetadataParameterItem {
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

export type FeralFileV1UpdateOperatorsParameter = FeralFileV1UpdateOperatorsParameterItem[];

export interface FeralFileV1UpdateOperatorsParameterItem {
    add_operator?: FeralFileV1UpdateOperatorsParameterItemAddOperator;

    remove_operator?: FeralFileV1UpdateOperatorsParameterItemRemoveOperator;
}

export interface FeralFileV1UpdateOperatorsParameterItemAddOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface FeralFileV1UpdateOperatorsParameterItemRemoveOperator {
    /** Tezos address. */
    operator: string;

    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;
}

export interface FeralFileV1CurrentStorage {
    admin: FeralFileV1CurrentStorageAdmin;

    /**
     * In-memory map.
     * 
     * Key of `string`: Bytes.
     * 
     * Value of `FeralFileV1CurrentStorageArtworksValue`.
     */
    artworks: MichelsonMap<string, FeralFileV1CurrentStorageArtworksValue>;

    assets: FeralFileV1CurrentStorageAssets;

    /** Simple boolean. */
    bridgeable: boolean;

    /** Simple boolean. */
    burnable: boolean;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `unknown`: Lambda.
     */
    bytes_utils: BigMapAbstraction;

    /** Arbitrary string. */
    exhibition_title: string;

    /** Nat - arbitrary big integer >= 0. */
    max_royalty_bps: BigNumber;

    /**
     * Big map.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    metadata: BigMapAbstraction;

    /** Nat - arbitrary big integer >= 0. */
    secondary_sale_royalty_bps: BigNumber;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `FeralFileV1CurrentStorageTokenAttributeValue`.
     */
    token_attribute: BigMapAbstraction;

    trustee: FeralFileV1CurrentStorageTrustee;
}

export interface FeralFileV1CurrentStorageAdmin {
    /** Tezos address. */
    admin: string;

    /** Tezos address. */
    pending_admin: string | null;
}

export interface FeralFileV1CurrentStorageArtworksValue {
    /** Arbitrary string. */
    artist_name: string;

    /** Bytes. */
    fingerprint: string;

    /** Nat - arbitrary big integer >= 0. */
    max_edition: BigNumber;

    /** Tezos address. */
    royalty_address: string;

    /** Arbitrary string. */
    title: string;

    /** Nat - arbitrary big integer >= 0. */
    token_start_id: BigNumber;
}

export interface FeralFileV1CurrentStorageAssets {
    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `string`: Tezos address.
     */
    ledger: BigMapAbstraction;

    /**
     * Big map.
     * 
     * Key of `FeralFileV1CurrentStorageAssetsOperatorsKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    operators: BigMapAbstraction;

    /**
     * Big map.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `FeralFileV1CurrentStorageAssetsTokenMetadataValue`.
     */
    token_metadata: BigMapAbstraction;
}

export interface FeralFileV1CurrentStorageAssetsOperatorsKey {
    /** Tezos address. */
    '0': string;

    /** Tezos address. */
    '1': string;

    /** Nat - arbitrary big integer >= 0. */
    '2': BigNumber;
}

export interface FeralFileV1CurrentStorageAssetsTokenMetadataValue {
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

export interface FeralFileV1CurrentStorageTokenAttributeValue {
    /** Bytes. */
    artwork_id: string;

    /** Simple boolean. */
    burned: boolean;

    /** Nat - arbitrary big integer >= 0. */
    edition_number: BigNumber;
}

export interface FeralFileV1CurrentStorageTrustee {
    /** Nat - arbitrary big integer >= 0. */
    max_trustee: BigNumber;

    /** Array of: Tezos address. */
    trustees: string[];
}

export interface FeralFileV1ChangedStorage {
    admin: FeralFileV1ChangedStorageAdmin;

    /**
     * In-memory map.
     * 
     * Key of `string`: Bytes.
     * 
     * Value of `FeralFileV1ChangedStorageArtworksValue`.
     */
    artworks: MichelsonMap<string, FeralFileV1ChangedStorageArtworksValue>;

    assets: FeralFileV1ChangedStorageAssets;

    /** Simple boolean. */
    bridgeable: boolean;

    /** Simple boolean. */
    burnable: boolean;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    bytes_utils: string;

    /** Arbitrary string. */
    exhibition_title: string;

    /** Nat - arbitrary big integer >= 0. */
    max_royalty_bps: BigNumber;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    metadata: string;

    /** Nat - arbitrary big integer >= 0. */
    secondary_sale_royalty_bps: BigNumber;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    token_attribute: string;

    trustee: FeralFileV1ChangedStorageTrustee;
}

export interface FeralFileV1ChangedStorageAdmin {
    /** Tezos address. */
    admin: string;

    /** Tezos address. */
    pending_admin: string | null;
}

export interface FeralFileV1ChangedStorageArtworksValue {
    /** Arbitrary string. */
    artist_name: string;

    /** Bytes. */
    fingerprint: string;

    /** Nat - arbitrary big integer >= 0. */
    max_edition: BigNumber;

    /** Tezos address. */
    royalty_address: string;

    /** Arbitrary string. */
    title: string;

    /** Nat - arbitrary big integer >= 0. */
    token_start_id: BigNumber;
}

export interface FeralFileV1ChangedStorageAssets {
    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    ledger: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    operators: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    token_metadata: string;
}

export interface FeralFileV1ChangedStorageTrustee {
    /** Nat - arbitrary big integer >= 0. */
    max_trustee: BigNumber;

    /** Array of: Tezos address. */
    trustees: string[];
}

export interface FeralFileV1InitialStorage {
    admin: FeralFileV1InitialStorageAdmin;

    /**
     * In-memory map.
     * 
     * Key of `string`: Bytes.
     * 
     * Value of `FeralFileV1InitialStorageArtworksValue`.
     */
    artworks: MichelsonMap<string, FeralFileV1InitialStorageArtworksValue>;

    assets: FeralFileV1InitialStorageAssets;

    /** Simple boolean. */
    bridgeable: boolean;

    /** Simple boolean. */
    burnable: boolean;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `unknown`: Lambda.
     */
    bytes_utils: MichelsonMap<BigNumber, unknown>;

    /** Arbitrary string. */
    exhibition_title: string;

    /** Nat - arbitrary big integer >= 0. */
    max_royalty_bps: BigNumber;

    /**
     * Big map initial values.
     * 
     * Key of `string`: Arbitrary string.
     * 
     * Value of `string`: Bytes.
     */
    metadata: MichelsonMap<string, string>;

    /** Nat - arbitrary big integer >= 0. */
    secondary_sale_royalty_bps: BigNumber;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `FeralFileV1InitialStorageTokenAttributeValue`.
     */
    token_attribute: MichelsonMap<BigNumber, FeralFileV1InitialStorageTokenAttributeValue>;

    trustee: FeralFileV1InitialStorageTrustee;
}

export interface FeralFileV1InitialStorageAdmin {
    /** Tezos address. */
    admin: string;

    /** Tezos address. */
    pending_admin: string | null;
}

export interface FeralFileV1InitialStorageArtworksValue {
    /** Arbitrary string. */
    artist_name: string;

    /** Bytes. */
    fingerprint: string;

    /** Nat - arbitrary big integer >= 0. */
    max_edition: BigNumber;

    /** Tezos address. */
    royalty_address: string;

    /** Arbitrary string. */
    title: string;

    /** Nat - arbitrary big integer >= 0. */
    token_start_id: BigNumber;
}

export interface FeralFileV1InitialStorageAssets {
    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `string`: Tezos address.
     */
    ledger: MichelsonMap<BigNumber, string>;

    /**
     * Big map initial values.
     * 
     * Key of `FeralFileV1InitialStorageAssetsOperatorsKey`.
     * 
     * Value of `typeof UnitValue`: An empty result.
     */
    operators: MichelsonMap<FeralFileV1InitialStorageAssetsOperatorsKey, typeof UnitValue>;

    /**
     * Big map initial values.
     * 
     * Key of `BigNumber`: Nat - arbitrary big integer >= 0.
     * 
     * Value of `FeralFileV1InitialStorageAssetsTokenMetadataValue`.
     */
    token_metadata: MichelsonMap<BigNumber, FeralFileV1InitialStorageAssetsTokenMetadataValue>;
}

export interface FeralFileV1InitialStorageAssetsOperatorsKey {
    /** Tezos address. */
    '0': string;

    /** Tezos address. */
    '1': string;

    /** Nat - arbitrary big integer >= 0. */
    '2': BigNumber;
}

export interface FeralFileV1InitialStorageAssetsTokenMetadataValue {
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

export interface FeralFileV1InitialStorageTokenAttributeValue {
    /** Bytes. */
    artwork_id: string;

    /** Simple boolean. */
    burned: boolean;

    /** Nat - arbitrary big integer >= 0. */
    edition_number: BigNumber;
}

export interface FeralFileV1InitialStorageTrustee {
    /** Nat - arbitrary big integer >= 0. */
    max_trustee: BigNumber;

    /** Array of: Tezos address. */
    trustees: string[];
}

/** Nat - arbitrary big integer >= 0. */
export type FeralFileV1AssetsLedgerKey = BigNumber;

/** Tezos address. */
export type FeralFileV1AssetsLedgerValue = string;

export interface FeralFileV1AssetsOperatorsKey {
    /** Tezos address. */
    '0': string;

    /** Tezos address. */
    '1': string;

    /** Nat - arbitrary big integer >= 0. */
    '2': BigNumber;
}

/** An empty result. */
export type FeralFileV1AssetsOperatorsValue = typeof UnitValue;

/** Nat - arbitrary big integer >= 0. */
export type FeralFileV1AssetsTokenMetadataKey = BigNumber;

export interface FeralFileV1AssetsTokenMetadataValue {
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
export type FeralFileV1BytesUtilsKey = BigNumber;

/** Lambda. */
export type FeralFileV1BytesUtilsValue = unknown;

/** Arbitrary string. */
export type FeralFileV1MetadataKey = string;

/** Bytes. */
export type FeralFileV1MetadataValue = string;

/** Nat - arbitrary big integer >= 0. */
export type FeralFileV1TokenAttributeKey = BigNumber;

export interface FeralFileV1TokenAttributeValue {
    /** Bytes. */
    artwork_id: string;

    /** Simple boolean. */
    burned: boolean;

    /** Nat - arbitrary big integer >= 0. */
    edition_number: BigNumber;
}
