// package: 
// file: event-processor.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

export class EventInput extends jspb.Message { 
    getType(): string;
    setType(value: string): EventInput;
    getBlockchain(): string;
    setBlockchain(value: string): EventInput;
    getContract(): string;
    setContract(value: string): EventInput;
    getFrom(): string;
    setFrom(value: string): EventInput;
    getTo(): string;
    setTo(value: string): EventInput;
    getTokenid(): string;
    setTokenid(value: string): EventInput;
    getTxid(): string;
    setTxid(value: string): EventInput;

    hasTxtime(): boolean;
    clearTxtime(): void;
    getTxtime(): google_protobuf_timestamp_pb.Timestamp | undefined;
    setTxtime(value?: google_protobuf_timestamp_pb.Timestamp): EventInput;
    getEventindex(): number;
    setEventindex(value: number): EventInput;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): EventInput.AsObject;
    static toObject(includeInstance: boolean, msg: EventInput): EventInput.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: EventInput, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): EventInput;
    static deserializeBinaryFromReader(message: EventInput, reader: jspb.BinaryReader): EventInput;
}

export namespace EventInput {
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
