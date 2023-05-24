// package: 
// file: event-processor.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as event_processor_pb from "./event-processor_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

interface IEventProcessorService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    pushEvent: IEventProcessorService_IPushEvent;
}

interface IEventProcessorService_IPushEvent extends grpc.MethodDefinition<event_processor_pb.EventInput, event_processor_pb.EventOutput> {
    path: "/EventProcessor/PushEvent";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<event_processor_pb.EventInput>;
    requestDeserialize: grpc.deserialize<event_processor_pb.EventInput>;
    responseSerialize: grpc.serialize<event_processor_pb.EventOutput>;
    responseDeserialize: grpc.deserialize<event_processor_pb.EventOutput>;
}

export const EventProcessorService: IEventProcessorService;

export interface IEventProcessorServer extends grpc.UntypedServiceImplementation {
    pushEvent: grpc.handleUnaryCall<event_processor_pb.EventInput, event_processor_pb.EventOutput>;
}

export interface IEventProcessorClient {
    pushEvent(request: event_processor_pb.EventInput, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    pushEvent(request: event_processor_pb.EventInput, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    pushEvent(request: event_processor_pb.EventInput, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
}

export class EventProcessorClient extends grpc.Client implements IEventProcessorClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public pushEvent(request: event_processor_pb.EventInput, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    public pushEvent(request: event_processor_pb.EventInput, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
    public pushEvent(request: event_processor_pb.EventInput, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: event_processor_pb.EventOutput) => void): grpc.ClientUnaryCall;
}
