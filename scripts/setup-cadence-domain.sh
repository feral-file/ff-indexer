#!/bin/bash

# Script to register Cadence domain for ff-indexer
# This should be run after Cadence is up and running

echo "🔧 Setting up Cadence domain for ff-indexer..."

# Wait for Cadence to be ready
echo "⏳ Waiting for Cadence to be ready..."
while ! docker exec ff-indexer-cadence cadence --address cadence:7933 cluster health >/dev/null 2>&1; do
  echo "   Cadence not ready yet, waiting..."
  sleep 2
done
echo "✅ Cadence is ready!"

# Register the domain
echo "📝 Registering 'nft-indexer' domain in Cadence..."
docker exec ff-indexer-cadence cadence --address cadence:7933 --domain nft-indexer domain register \
  --retention 7 \
  --description "FF-Indexer NFT processing workflows"

if [ $? -eq 0 ]; then
    echo "✅ Cadence domain 'nft-indexer' registered successfully!"
else
    echo "⚠️  Domain registration failed or domain already exists"
fi

echo "🎉 Cadence setup complete!"
echo "📍 Access Cadence Web UI at: http://localhost:8088"
