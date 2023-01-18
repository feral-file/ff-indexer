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

const eventProcessorUri = <string>process.env.EVENT_PROCESSOR_URL
const eventSubscriberUrl = <string>process.env.EVENT_SUBSCRIBER_URL

if (!eventSubscriberUrl) {
  console.log("[TEZOS_MARLETPLACE_INDEXER]", "event subscriber url not set")
}

var grpcClient: EventProcessorClient
if (eventProcessorUri) {
  grpcClient = new EventProcessorClient(eventProcessorUri, grpc.ChannelCredentials.createInsecure());
} else {
  console.log("[TEZOS_MARLETPLACE_INDEXER]", "event processor uri not set")
}

async function reportEvent(timestamp: Date, contract: string, tokenID: string, from_: string, to_: string) {
  await axios.post(eventSubscriberUrl, {
    "timestamp": timestamp,
    "contract": contract,
    "tokenID": tokenID,
    "from": from_,
    "to": to_,
    "isTest": IS_TESTNET
  })
}

async function grpcReportEvent(timestamp: Date, contract: string, tokenID: string, from_: string, to_: string) {
  if (!grpcClient) {
    throw Error("grpc client is not initialized")
  }

  let event = new EventInput()
  event.setBlockchain("tezos")
  event.setEventtype("transfer")
  event.setContract(contract)
  event.setFrom(from_)
  event.setTo(to_)
  event.setTokenid(tokenID)
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
          "id:", items.token_id.toFixed(), "from:", transfer.from_, "to:", items.to_)
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
        await grpcReportEvent(
          indexingContext.block.timestamp, indexingContext.contract.address,
          items.token_id.toFixed(), transfer.from_, items.to_)
        console.log("[TOKEN_TRANSFER]", "<GRPC>", "(", indexingContext.contract.address, ")",
          "id:", items.token_id.toFixed(), "from:", transfer.from_, "to:", items.to_)
      } catch (error) {
        console.log("fail to push event through grpc", error)
      }
    })
  }))
}


export function outputTransferAPI(parameter: TezosTransferParameterItem[], indexingContext: TransactionIndexingContext) {
  parameter.forEach((transfer => {
    transfer.txs.forEach(async function (items) {
      try {
        await reportEvent(
          indexingContext.block.timestamp, indexingContext.contract.address,
          items.token_id.toFixed(), transfer.from_, items.to_)
        console.log("[TOKEN_TRANSFER]", "<API>", "(", indexingContext.contract.address, ")",
          "id:", items.token_id.toFixed(), "from:", transfer.from_, "to:", items.to_)
      } catch (error) {
        console.log("fail to push event through api", error)
      }
    })
  }))
}

export function outputTransfer(parameter: TezosTransferParameterItem[], indexingContext: TransactionIndexingContext) {
  if (eventProcessorUri) {
    outputTransferGRPC(parameter, indexingContext)
  }

  if (eventSubscriberUrl) {
    outputTransferAPI(parameter, indexingContext)
  }

  if (!eventProcessorUri && !eventSubscriberUrl) {
    outputTransferStdout(parameter, indexingContext)
  }
}
