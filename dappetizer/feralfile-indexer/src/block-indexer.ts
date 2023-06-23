import { DbContext } from '@tezos-dappetizer/database';

import {
    Block,
    BlockIndexingContext,
} from '@tezos-dappetizer/indexer';

const AWS = require('aws-sdk')

const ssmClient = new AWS.SSM({
    apiVersion: '2014-11-06',
    region: <string>process.env.AWS_REGION
});

const lastStopBlockKeyName = <string>process.env.LAST_STOP_BLOCK_KEY_NAME;

export class BlockDataIndexer {
    async indexBlock(block: Block, dbContext: DbContext, indexingContext: BlockIndexingContext): Promise<void> {
        if (block.level % 5 != 0) {
            return
        }
        console.log("update last block", block.level)
        try {
            await ssmClient.putParameter({
                Name: lastStopBlockKeyName,
                Type: "String",
                Value: "" + block.level,
                Overwrite: true
            }).promise()
        } catch (error) {
            console.log(error)
        }
    }
}
