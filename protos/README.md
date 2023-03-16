# gRPC



## Getting started

- Follow this link: https://medium.com/@tj.ruesch/your-first-grpc-api-in-golang-d277d684b84e

## Installation
- Macos:
  ```brew install protobuf```

- ubuntu:
  ```apt install -y protobuf-compiler```

- install _protoc-gen-go_ and _protoc-gen-go-grpc_
  ```
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
  ```
  
- move to nft-event-processor/protos folder:
  ```
    cd protos
  ```
  
- generate protobuf golang code:
    ```
    protoc --go-grpc_out=../services/nft-event-processor/ --go_out=../services/nft-event-processor/ event-processor.proto
    protoc --go-grpc_out=../services/nft-indexer-grpc/ --go_out=../services/nft-indexer-grpc/ indexer.proto
    ```
  
- generate protobuf js code:
  - Create npm package and install tools, folder
      ```
      npm init
      npm install grpc-tools ts-protoc-gen
      mkdir ./src/grpc
      ```
  - generate js and ts code example:
      ```
      protoc \
          --plugin="protoc-gen-ts=./node_modules/.bin/protoc-gen-ts" \
          --plugin="protoc-gen-grpc=./node_modules/.bin/grpc_tools_node_protoc_plugin" \
	  --js_out="import_style=commonjs,binary:./src/grpc" \
          --ts_out="service=grpc-node:./src/grpc" \
          --grpc_out="./src/grpc" \
	  --proto_path="../../protos/" \
         event-processor.proto
      ```



