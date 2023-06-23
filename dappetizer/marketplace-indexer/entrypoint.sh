#!/bin/sh

rm database.sqlite

[ -f ".env" ] && source .env

export NETWORK=${NETWORK:-livenet}
export TEZOS_NODE_RPC_URL=${TEZOS_NODE_RPC_URL:-https://mainnet-tezos.autonomy.io}

# Get last stop block from AWS
aws configure --profile autonomy-event-emitter set aws_access_key_id $AWS_ACCESS_KEY_ID
aws configure --profile autonomy-event-emitter set aws_secret_access_key $AWS_SECRET_ACCESS_KEY
aws configure --profile autonomy-event-emitter set region $AWS_REGION
export LAST_STOP_BLOCK=$(aws ssm --profile autonomy-event-emitter get-parameter --name "$LAST_STOP_BLOCK_KEY_NAME" | jq -r .Parameter.Value)
export INDEXER_START_BLOCK=${INDEXER_START_BLOCK:-$LAST_STOP_BLOCK}

# Get currnet block from RPC
export CURRENT_LAST_BLOCK=$(curl $TEZOS_NODE_RPC_URL/chains/main/blocks/head/header | jq .level)
export INDEXER_START_BLOCK=${INDEXER_START_BLOCK:-$CURRENT_LAST_BLOCK}

export DEFAULT_CONFIG_PATH=${DEFAULT_CONFIG_PATH:-/.config/dappetizer.default.config.json}

jq --arg blockNumber "$INDEXER_START_BLOCK" --arg tezosRPCNodeURL "$TEZOS_NODE_RPC_URL" \
    '.indexing.fromBlockLevel = ($blockNumber|tonumber) | .tezosNode.url = $tezosRPCNodeURL' $DEFAULT_CONFIG_PATH > dappetizer.mainnet.config.json

./node_modules/.bin/dappetizer start
