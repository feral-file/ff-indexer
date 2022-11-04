#!/bin/sh

rm database.sqlite

[ -f ".env" ] && source .env

export NETWORK=${NETWORK:-livenet}
export EVENT_PROCESSOR_URL=${EVENT_PROCESSOR_URL:-http://localhost:8089/events}
export TEZOS_NODE_RPC_URL=${TEZOS_NODE_RPC_URL:-https://mainnet-tezos.autonomy.io}

export CURRENT_LAST_BLOCK=$(curl $TEZOS_NODE_RPC_URL/chains/main/blocks/head/header | jq .level)
export INDEXER_START_BLOCK=${INDEXER_START_BLOCK:-$CURRENT_LAST_BLOCK}

export DEFAULT_CONFIG_PATH=${DEFAULT_CONFIG_PATH:-/.config/dappetizer.default.config.json}

jq --arg blockNumber "$INDEXER_START_BLOCK" --arg tezosRPCNodeURL "$TEZOS_NODE_RPC_URL" \
    '.indexing.fromBlockLevel = ($blockNumber|tonumber) | .tezosNode.url = $tezosRPCNodeURL' $DEFAULT_CONFIG_PATH > dappetizer.mainnet.config.json

./node_modules/.bin/dappetizer start
