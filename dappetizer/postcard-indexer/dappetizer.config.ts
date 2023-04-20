import { DappetizerConfigUsingDb } from '@tezos-dappetizer/database';
import { loadDappetizerNetworkConfigs } from '@tezos-dappetizer/indexer';

const config: DappetizerConfigUsingDb = {
    modules: [{
        id: '.', // This project is the indexer module itself.
    }],
    networks: loadDappetizerNetworkConfigs(__dirname),
    database: {
        type: 'sqlite',
        database: 'database.sqlite',

        // If you want to use PostgreSQL:
        // type: 'postgres',
        // host: 'localhost',
        // port: 5432,
        // username: 'postgres',
        // password: 'postgrespassword',
        // database: 'postgres',
        // schema: 'indexer',
    },
    usageStatistics: {
        enabled: true,
    },
};

export default config;
