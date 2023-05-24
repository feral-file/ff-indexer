import {
  BigNumber
} from 'bignumber.js';
import {
  TransactionIndexingContext,
} from '@tezos-dappetizer/indexer';
import axios from 'axios';

import { EventProcessorClient } from './event-processor_grpc_pb'
import { EventInput } from './event-processor_pb'
import * as grpc from "@grpc/grpc-js";
import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb';

interface TransferParameterItemTxsItem {
  /** Nat - arbitrary big integer >= 0. */
  amount: BigNumber;
  /** Tezos address. */
  to_: string;
  /** Nat - arbitrary big integer >= 0. */
  token_id: BigNumber;
}

interface TezosTransferParameterItem {
  /** Tezos address. */
  from_: string;
  txs: TransferParameterItemTxsItem[];
}

const IS_TESTNET = <string>process.env.NETWORK == "testnet"

const eventProcessorURI = <string>process.env.EVENT_PROCESSOR_URI

var grpcClient: EventProcessorClient
if (eventProcessorURI) {
  grpcClient = new EventProcessorClient(eventProcessorURI, grpc.ChannelCredentials.createInsecure());
} else {
  console.log("[TEZOS_EMITTER]", "event processor uri not set")
}

async function grpcReportEvent(timestamp: Date, contract: string, tokenID: string, from_: string, to_: string, txID: string, txTime: Date) {
  if (!grpcClient) {
    throw Error("grpc client is not initialized")
  }

  let event = new EventInput()
  event.setBlockchain("tezos")
  event.setType("transfer")
  event.setContract(contract)
  event.setFrom(from_)
  event.setTo(to_)
  event.setTokenid(tokenID)
  event.setTxid(txID)
  event.setTxtime(Timestamp.fromDate(txTime))
  await new Promise((resolve, reject) => {
    grpcClient.pushEvent(event, (error, response) => {
      if (error) {
        reject(error);
        return
      }

      let respStatus = response.getStatus()
      if (respStatus == 200) {
        resolve("")
      } else {
        reject(response.getResult());
      }
    })
  })
}

export function outputTransferStdout(parameter: TezosTransferParameterItem[], indexingContext: TransactionIndexingContext) {
  parameter.forEach((transfer => {
    transfer.txs.forEach(async function (items) {
      try {
        console.log("[TOKEN_TRANSFER]", "<STDOUT>", "(", indexingContext.contract.address, ")",
          "id:", items.token_id.toFixed(), "from:", transfer.from_, "to:", items.to_,
          "txid:", indexingContext.operationGroup.hash, "txTime:", indexingContext.block.timestamp
        )
      } catch (error) {
        console.log("fail to push event", error)
        throw error
      }
    })
  }))
}

export function outputTransferGRPC(parameter: TezosTransferParameterItem[], indexingContext: TransactionIndexingContext) {
  parameter.forEach((transfer => {
    transfer.txs.forEach(async function (items) {
      try {
        console.log("[TOKEN_TRANSFER]", "<GRPC>", "(", indexingContext.contract.address, ")",
          "id:", items.token_id.toFixed(), "from:", transfer.from_, "to:", items.to_,
          "txid:", indexingContext.operationGroup.hash, "txTime:", indexingContext.block.timestamp
        )
        await grpcReportEvent(
          indexingContext.block.timestamp, indexingContext.contract.address,
          items.token_id.toFixed(), transfer.from_, items.to_,
          indexingContext.operationGroup.hash, indexingContext.block.timestamp)
      } catch (error) {
        console.log("fail to push event through grpc", error)
      }
    })
  }))
}

export function outputTransfer(parameter: TezosTransferParameterItem[], indexingContext: TransactionIndexingContext) {
  if (eventProcessorURI) {
    outputTransferGRPC(parameter, indexingContext)
  } else {
    outputTransferStdout(parameter, indexingContext)
  }
}
