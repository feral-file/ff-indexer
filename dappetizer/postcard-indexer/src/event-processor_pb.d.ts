// package: 
// file: event-processor.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";
import * as google_protobuf_struct_pb from "google-protobuf/google/protobuf/struct_pb";

export class NftEventInput extends jspb.Message { 
    getType(): string;
    setType(value: string): NftEventInput;
    getBlockchain(): string;
    setBlockchain(value: string): NftEventInput;
    getContract(): string;
    setContract(value: string): NftEventInput;
    getFrom(): string;
    setFrom(value: string): NftEventInput;
    getTo(): string;
    setTo(value: string): NftEventInput;
    getTokenid(): string;
    setTokenid(value: string): NftEventInput;
    getTxid(): string;
    setTxid(value: string): NftEventInput;

    hasTxtime(): boolean;
    clearTxtime(): void;
    getTxtime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setTxtime(value?: google_protobuf_timestamp_pb.Timestamp): NftEventInput;
    getEventindex(): number;
    setEventindex(value: number): NftEventInput;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): NftEventInput.AsObject;
    static toObject(includeInstance: boolean, msg: NftEventInput): NftEventInput.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: NftEventInput, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): NftEventInput;
    static deserializeBinaryFromReader(message: NftEventInput, reader: jspb.BinaryReader): NftEventInput;
}

export namespace NftEventInput {
    export type AsObject = {
        type: string,
        blockchain: string,
        contract: string,
        from: string,
        to: string,
        tokenid: string,
        txid: string,
        txtime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        eventindex: number,
    }
}

export class SeriesEventInput extends jspb.Message { 
    getType(): string;
    setType(value: string): SeriesEventInput;
    getContract(): string;
    setContract(value: string): SeriesEventInput;

    hasData(): boolean;
    clearData(): void;
    getData(): google_protobuf_struct_pb.Struct | undefined;
    setData(value?: google_protobuf_struct_pb.Struct): SeriesEventInput;
    getTxid(): string;
    setTxid(value: string): SeriesEventInput;

    hasTxtime(): boolean;
    clearTxtime(): void;
    getTxtime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setTxtime(value?: google_protobuf_timestamp_pb.Timestamp): SeriesEventInput;
    getEventindex(): number;
    setEventindex(value: number): SeriesEventInput;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): SeriesEventInput.AsObject;
    static toObject(includeInstance: boolean, msg: SeriesEventInput): SeriesEventInput.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: SeriesEventInput, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): SeriesEventInput;
    static deserializeBinaryFromReader(message: SeriesEventInput, reader: jspb.BinaryReader): SeriesEventInput;
}

export namespace SeriesEventInput {
    export type AsObject = {
        type: string,
        contract: string,
        data?: google_protobuf_struct_pb.Struct.AsObject,
        txid: string,
        txtime?: google_protobuf_timestamp_pb.Timestamp.AsObject,
        eventindex: number,
    }
}

export class EventOutput extends jspb.Message { 
    getResult(): string;
    setResult(value: string): EventOutput;
    getStatus(): number;
    setStatus(value: number): EventOutput;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): EventOutput.AsObject;
    static toObject(includeInstance: boolean, msg: EventOutput): EventOutput.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: EventOutput, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): EventOutput;
    static deserializeBinaryFromReader(message: EventOutput, reader: jspb.BinaryReader): EventOutput;
}

export namespace EventOutput {
    export type AsObject = {
        result: string,
        status: number,
    }
}
