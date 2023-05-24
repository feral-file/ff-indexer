/* istanbul ignore next */
/* eslint-disable */

// This file was generated.
// It should NOT be modified manually rather it should be regenerated.
// Contract: KT1MDvWtwi8sCcyJdbWPScTdFa2uJ8mnKNJe
// Tezos network: ghostnet

import { UnitValue } from '@taquito/michelson-encoder';
import { BigMapAbstraction, MichelsonMap } from '@taquito/taquito';
import { BigNumber } from 'bignumber.js';

export type PostcardParameter =
    | { entrypoint: 'add_signer'; value: PostcardAddSignerParameter }
    | { entrypoint: 'admin'; value: PostcardAdminParameter }
    | { entrypoint: 'assets'; value: PostcardAssetsParameter }
    | { entrypoint: 'balance_of'; value: PostcardBalanceOfParameter }
    | { entrypoint: 'confirm_admin'; value: PostcardConfirmAdminParameter }
    | { entrypoint: 'mail_postcard'; value: PostcardMailPostcardParameter }
    | { entrypoint: 'mint_postcard'; value: PostcardMintPostcardParameter }
    | { entrypoint: 'pause'; value: PostcardPauseParameter }
    | { entrypoint: 'postcards'; value: PostcardPostcardsParameter }
    | { entrypoint: 'remove_signer'; value: PostcardRemoveSignerParameter }
    | { entrypoint: 'set_admin'; value: PostcardSetAdminParameter }
    | { entrypoint: 'signer'; value: PostcardSignerParameter }
    | { entrypoint: 'stamp_postcard'; value: PostcardStampPostcardParameter }
    | { entrypoint: 'transfer'; value: PostcardTransferParameter }
    | { entrypoint: 'update_operators'; value: PostcardUpdateOperatorsParameter };

/** Tezos address. */
export type PostcardAddSignerParameter = string;

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

export interface PostcardMailPostcardParameter {
    params: PostcardMailPostcardParameterParamsItem[];

    signer: PostcardMailPostcardParameterSigner;
}

export interface PostcardMailPostcardParameterParamsItem {
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

export interface PostcardMailPostcardParameterSigner {
    /** Key. */
    pk: string;

    /** Signature. */
    signature: string;

    /** Date ISO 8601 string. */
    timestamp: string;
}

export interface PostcardMintPostcardParameter {
    params: PostcardMintPostcardParameterParamsItem[];

    signer: PostcardMintPostcardParameterSigner;
}

export interface PostcardMintPostcardParameterParamsItem {
    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;

    /** Bytes. */
    token_info_uri: string;
}

export interface PostcardMintPostcardParameterSigner {
    /** Key. */
    pk: string;

    /** Signature. */
    signature: string;

    /** Date ISO 8601 string. */
    timestamp: string;
}

/** Simple boolean. */
export type PostcardPauseParameter = boolean;

export interface PostcardPostcardsParameter {
    mail_postcard?: PostcardPostcardsParameterMailPostcard;

    mint_postcard?: PostcardPostcardsParameterMintPostcard;

    stamp_postcard?: PostcardPostcardsParameterStampPostcard;
}

export interface PostcardPostcardsParameterMailPostcard {
    params: PostcardPostcardsParameterMailPostcardParamsItem[];

    signer: PostcardPostcardsParameterMailPostcardSigner;
}

export interface PostcardPostcardsParameterMailPostcardParamsItem {
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

export interface PostcardPostcardsParameterMailPostcardSigner {
    /** Key. */
    pk: string;

    /** Signature. */
    signature: string;

    /** Date ISO 8601 string. */
    timestamp: string;
}

export interface PostcardPostcardsParameterMintPostcard {
    params: PostcardPostcardsParameterMintPostcardParamsItem[];

    signer: PostcardPostcardsParameterMintPostcardSigner;
}

export interface PostcardPostcardsParameterMintPostcardParamsItem {
    /** Tezos address. */
    owner: string;

    /** Nat - arbitrary big integer >= 0. */
    token_id: BigNumber;

    /** Bytes. */
    token_info_uri: string;
}

export interface PostcardPostcardsParameterMintPostcardSigner {
    /** Key. */
    pk: string;

    /** Signature. */
    signature: string;

    /** Date ISO 8601 string. */
    timestamp: string;
}

export interface PostcardPostcardsParameterStampPostcard {
    params: PostcardPostcardsParameterStampPostcardParamsItem[];

    signer: PostcardPostcardsParameterStampPostcardSigner;
}

export interface PostcardPostcardsParameterStampPostcardParamsItem {
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

export interface PostcardPostcardsParameterStampPostcardSigner {
    /** Key. */
    pk: string;

    /** Signature. */
    signature: string;

    /** Date ISO 8601 string. */
    timestamp: string;
}

/** Tezos address. */
export type PostcardRemoveSignerParameter = string;

/** Tezos address. */
export type PostcardSetAdminParameter = string;

export interface PostcardSignerParameter {
    /** Tezos address. */
    add_signer?: string;

    /** Tezos address. */
    remove_signer?: string;
}

export interface PostcardStampPostcardParameter {
    params: PostcardStampPostcardParameterParamsItem[];

    signer: PostcardStampPostcardParameterSigner;
}

export interface PostcardStampPostcardParameterParamsItem {
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

export interface PostcardStampPostcardParameterSigner {
    /** Key. */
    pk: string;

    /** Signature. */
    signature: string;

    /** Date ISO 8601 string. */
    timestamp: string;
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

    signers: PostcardCurrentStorageSigners;
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

export interface PostcardCurrentStorageSigners {
    /** Nat - arbitrary big integer >= 0. */
    max_signer: BigNumber;

    /** Array of: Tezos address. */
    signers: string[];
}

export interface PostcardChangedStorage {
    admin: PostcardChangedStorageAdmin;

    assets: PostcardChangedStorageAssets;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    metadata: string;

    /** Big map ID - string with arbitrary big integer, negative if temporary. */
    postcards: string;

    signers: PostcardChangedStorageSigners;
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

export interface PostcardChangedStorageSigners {
    /** Nat - arbitrary big integer >= 0. */
    max_signer: BigNumber;

    /** Array of: Tezos address. */
    signers: string[];
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

    signers: PostcardInitialStorageSigners;
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

export interface PostcardInitialStorageSigners {
    /** Nat - arbitrary big integer >= 0. */
    max_signer: BigNumber;

    /** Array of: Tezos address. */
    signers: string[];
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
