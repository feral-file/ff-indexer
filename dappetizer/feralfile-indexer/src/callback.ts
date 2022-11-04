import {
  BigNumber
} from 'bignumber.js';
import {
  TransactionIndexingContext,
} from '@tezos-dappetizer/indexer';
import axios from 'axios';

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

async function reportEvent(timestamp: Date, contract: string, tokenID: string, from_: string, to_: string) {
  await axios.post(<string>process.env.EVENT_PROCESSOR_URL, {
    "timestamp": timestamp,
    "contract": contract,
    "tokenID": tokenID,
    "from": from_,
    "to": to_,
    "isTest": IS_TESTNET
  })
}

export function outputTransferStdout(parameter: TezosTransferParameterItem[], indexingContext: TransactionIndexingContext) {
  {
    parameter.forEach((transfer => {
      transfer.txs.forEach(async function (items) {
        try {
          console.log("[TOKEN_TRANSFER]", "(", indexingContext.contract.address, ")",
            "id:", items.token_id.toFixed(), "from:", transfer.from_, "to:", items.to_)
        } catch (error) {
          console.log("fail to push event", error)
          throw error
        }
      })
    }))
  }
}

export function outputTransfer(parameter: TezosTransferParameterItem[], indexingContext: TransactionIndexingContext) {
  {
    parameter.forEach((transfer => {
      transfer.txs.forEach(async function (items) {
        try {
          await reportEvent(
            indexingContext.block.timestamp, indexingContext.contract.address,
            items.token_id.toFixed(), transfer.from_, items.to_)
          console.log("[TOKEN_TRANSFER]", "(", indexingContext.contract.address, ")",
            "id:", items.token_id.toFixed(), "from:", transfer.from_, "to:", items.to_)
        } catch (error) {
          console.log("fail to push event", error)
        }
      })
    }))
  }
}
