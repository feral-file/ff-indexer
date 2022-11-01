import { IndexerModuleUsingDb } from '@tezos-dappetizer/database';
import { createContractIndexerFromDecorators } from '@tezos-dappetizer/decorators';

import { FeralFileV1Indexer } from './feral-file-v-1-indexer';

export const indexerModule: IndexerModuleUsingDb = {
    name: 'NftIndexerTezosFeralfileIndexer',
    dbEntities: [
        // Register your DB entity classes to TypeORM here:
        // MyDbEntity,
    ],
    contractIndexers: [
        // Create your contract indexers here:
        createContractIndexerFromDecorators(new FeralFileV1Indexer()),
    ],
    blockDataIndexers: [
        // Create your block data indexers here:
        // new MyBlockDataIndexer(),
    ],
    // Create your indexing cycle handler here:
    // indexingCycleHandler: new MyIndexingCycleHandler(),
};
