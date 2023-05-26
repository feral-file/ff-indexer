// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var event$processor_pb = require('./event-processor_pb.js');
var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');

function serialize_EventInput(arg) {
  if (!(arg instanceof event$processor_pb.EventInput)) {
    throw new Error('Expected argument of type EventInput');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_EventInput(buffer_arg) {
  return event$processor_pb.EventInput.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_EventOutput(arg) {
  if (!(arg instanceof event$processor_pb.EventOutput)) {
    throw new Error('Expected argument of type EventOutput');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_EventOutput(buffer_arg) {
  return event$processor_pb.EventOutput.deserializeBinary(new Uint8Array(buffer_arg));
}


var EventProcessorService = exports.EventProcessorService = {
  pushEvent: {
    path: '/EventProcessor/PushEvent',
    requestStream: false,
    responseStream: false,
    requestType: event$processor_pb.EventInput,
    responseType: event$processor_pb.EventOutput,
    requestSerialize: serialize_EventInput,
    requestDeserialize: deserialize_EventInput,
    responseSerialize: serialize_EventOutput,
    responseDeserialize: deserialize_EventOutput,
  },
};

exports.EventProcessorClient = grpc.makeGenericClientConstructor(EventProcessorService);
