// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var event$processor_pb = require('./event-processor_pb.js');
var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');
var google_protobuf_struct_pb = require('google-protobuf/google/protobuf/struct_pb.js');

function serialize_EventOutput(arg) {
  if (!(arg instanceof event$processor_pb.EventOutput)) {
    throw new Error('Expected argument of type EventOutput');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_EventOutput(buffer_arg) {
  return event$processor_pb.EventOutput.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_NftEventInput(arg) {
  if (!(arg instanceof event$processor_pb.NftEventInput)) {
    throw new Error('Expected argument of type NftEventInput');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_NftEventInput(buffer_arg) {
  return event$processor_pb.NftEventInput.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_SeriesEventInput(arg) {
  if (!(arg instanceof event$processor_pb.SeriesEventInput)) {
    throw new Error('Expected argument of type SeriesEventInput');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_SeriesEventInput(buffer_arg) {
  return event$processor_pb.SeriesEventInput.deserializeBinary(new Uint8Array(buffer_arg));
}


var EventProcessorService = exports.EventProcessorService = {
  pushNftEvent: {
    path: '/EventProcessor/PushNftEvent',
    requestStream: false,
    responseStream: false,
    requestType: event$processor_pb.NftEventInput,
    responseType: event$processor_pb.EventOutput,
    requestSerialize: serialize_NftEventInput,
    requestDeserialize: deserialize_NftEventInput,
    responseSerialize: serialize_EventOutput,
    responseDeserialize: deserialize_EventOutput,
  },
  pushSeriesEvent: {
    path: '/EventProcessor/PushSeriesEvent',
    requestStream: false,
    responseStream: false,
    requestType: event$processor_pb.SeriesEventInput,
    responseType: event$processor_pb.EventOutput,
    requestSerialize: serialize_SeriesEventInput,
    requestDeserialize: deserialize_SeriesEventInput,
    responseSerialize: serialize_EventOutput,
    responseDeserialize: deserialize_EventOutput,
  },
};

exports.EventProcessorClient = grpc.makeGenericClientConstructor(EventProcessorService);
