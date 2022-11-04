import { IndexerModuleUsingDb } from '@tezos-dappetizer/database';
import { createContractIndexerFromDecorators } from '@tezos-dappetizer/decorators';

import { FxhashIndexer } from './fxhash-indexer';
import { Fxhashv2Indexer } from './fxhashv-2-indexer';
import { HicetniucIndexer } from './hicetniuc-indexer';
import { VersumIndexer } from './versum-indexer';

export const indexerModule: IndexerModuleUsingDb = {
    name: 'NftIndexerTezosFeralfileIndexerNew',
    dbEntities: [
        // Register your DB entity classes to TypeORM here:
        // MyDbEntity,
    ],
    contractIndexers: [
        // Create your contract indexers here:
        createContractIndexerFromDecorators(new FxhashIndexer()),
        createContractIndexerFromDecorators(new Fxhashv2Indexer()),
        createContractIndexerFromDecorators(new HicetniucIndexer()),
        createContractIndexerFromDecorators(new VersumIndexer()),
    ],
    blockDataIndexers: [
        // Create your block data indexers here:
        // new MyBlockDataIndexer(),
    ],
    // Create your indexing cycle handler here:
    // indexingCycleHandler: new MyIndexingCycleHandler(),
};
