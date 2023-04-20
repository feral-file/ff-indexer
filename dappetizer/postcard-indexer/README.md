# Autonomy NFT Event Indexer

This project scans each tezos blocks and handles various of blockchain changes separately.

## Pre-requisite

- node 16+
- npm 8+
- dappetizer 2.0.1+
- protobuf
    - @grpc/grpc-js


## Settings

Set `compilerOptions.allowJs = true` in `tsconfig.json`

## Generate grpc code

```
./node_modules/.bin/grpc_tools_node_protoc \
    --plugin="protoc-gen-ts=./node_modules/.bin/protoc-gen-ts" \
    --plugin="protoc-gen-grpc=./node_modules/.bin/grpc_tools_node_protoc_plugin" \
    --js_out="import_style=commonjs,binary:./src" \
    --ts_out="service=grpc-node,mode=grpc-js:./src" \
    --grpc_out="grpc_js:./src" \
    --proto_path="../../protos/" \
    event-processor.proto
```

## Build and Run

Update gRPC related files if any

```
npm run build-grpc
```

For any changes happen, we do

```
npm run build
```

Before run, we need to setup environment properly by copy and update `.env.sample` to `.env`. Then run

```
source .env
```

Finally, start the server

```
npx dappetizer start
```

## Add a new contract

First, use the command to generate indexer class

```
npx dappetizer update fxhashv2 KT1U6EHmNxJTkvaWJ4ThczG4FSDaHC21ssvi
```

After initiation,
- update callbacks to related entrypoint or bigmap change in the class.
- register the class in `index.ts`.
- update `dappetizer.config` with indexing contracts correctly configured.
- rebuild the indexer code using `npm run build`

Finally, we can start the indexer again by `npm run start`.


## Init
