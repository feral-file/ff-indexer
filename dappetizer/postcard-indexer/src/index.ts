import { IndexerModuleUsingDb } from '@tezos-dappetizer/database';
import { createContractIndexerFromDecorators } from '@tezos-dappetizer/decorators';

import { PostcardIndexer } from './postcard-indexer';

export const indexerModule: IndexerModuleUsingDb = {
    name: 'MyModule',
    dbEntities: [
        // Register your DB entity classes to TypeORM here:
        // MyDbEntity,
    ],
    contractIndexers: [
        // Create your contract indexers here:
        createContractIndexerFromDecorators(new PostcardIndexer()),
    ],
    blockDataIndexers: [
        // Create your block data indexers here:
        // new MyBlockDataIndexer(),
    ],
    // Create your indexing cycle handler here:
    // indexingCycleHandler: new MyIndexingCycleHandler(),
};
