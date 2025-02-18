// package: 
// file: event-processor.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as event_processor_pb from "./event-processor_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";
import * as google_protobuf_struct_pb from "google-protobuf/google/protobuf/struct_pb";

interface IEventProcessorService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    pushNftEvent: IEventProcessorService_IPushNftEvent;
    pushSeriesEvent: IEventProcessorService_IPushSeriesEvent;
}

interface IEventProcessorService_IPushNftEvent extends grpc.MethodDefinition<event_processor_pb.NftEventInput, event_processor_pb.EventOutput> {
    path: "/EventProcessor/PushNftEvent";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<event_processor_pb.NftEventInput>;
    requestDeserialize: grpc.deserialize<event_processor_pb.NftEventInput>;
    responseSerialize: grpc.serialize<event_processor_pb.EventOutput>;
    responseDeserialize: grpc.deserialize<event_processor_pb.EventOutput>;
}
interface IEventProcessorService_IPushSeriesEvent extends grpc.MethodDefinition<event_processor_pb.SeriesEventInput, event_processor_pb.EventOutput> {
    path: "/EventProcessor/PushSeriesEvent";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<event_processor_pb.SeriesEventInput>;
    requestDeserialize: grpc.deserialize<event_processor_pb.SeriesEventInput>;
    responseSerialize: grpc.serialize<event_processor_pb.EventOutput>;
    responseDeserialize: grpc.deserialize<event_processor_pb.EventOutput>;
}

export const EventProcessorService: IEventProcessorService;

export interface IEventProcessorServer extends grpc.UntypedServiceImplementation {
    pushNftEvent: grpc.handleUnaryCall<event_processor_pb.NftEventInput, event_processor_pb.EventOutput>;
    pushSeriesEvent: grpc.handleUnaryCall<event_processor_pb.SeriesEventInput, event_processor_pb.EventOutput>;
}

export interface IEventProcessorClient {
    pushNftEvent(request: event_processor_pb.NftEventInput, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    pushNftEvent(request: event_processor_pb.NftEventInput, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    pushNftEvent(request: event_processor_pb.NftEventInput, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    pushSeriesEvent(request: event_processor_pb.SeriesEventInput, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    pushSeriesEvent(request: event_processor_pb.SeriesEventInput, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    pushSeriesEvent(request: event_processor_pb.SeriesEventInput, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
}

export class EventProcessorClient extends grpc.Client implements IEventProcessorClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public pushNftEvent(request: event_processor_pb.NftEventInput, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    public pushNftEvent(request: event_processor_pb.NftEventInput, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    public pushNftEvent(request: event_processor_pb.NftEventInput, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    public pushSeriesEvent(request: event_processor_pb.SeriesEventInput, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    public pushSeriesEvent(request: event_processor_pb.SeriesEventInput, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    public pushSeriesEvent(request: event_processor_pb.SeriesEventInput, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
}
