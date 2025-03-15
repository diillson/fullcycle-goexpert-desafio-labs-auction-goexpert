#!/bin/bash
set -e

echo "Building and starting test containers..."
docker-compose -f docker-compose.test.yml up --build -d mongodb-test

echo "Waiting for MongoDB to initialize..."
sleep 5

echo "===================================="
echo "Running auction automatic closing test..."
echo "===================================="
docker-compose -f docker-compose.test.yml run --rm \
  -e AUCTION_INTERVAL=3s \
  -e AUCTION_CHECK_INTERVAL=1s \
  app-test sh -c "cd /app && go test -v ./internal/infra/database/auction -run TestAutomaticAuctionClosing"

echo "===================================="
echo "Running auction lifecycle integration test..."
echo "===================================="
docker-compose -f docker-compose.test.yml run --rm \
  -e AUCTION_INTERVAL=3s \
  -e AUCTION_CHECK_INTERVAL=1s \
  app-test sh -c "cd /app && go test -v ./internal/test/integration -run TestSimpleAuctionLifecycleIntegration"

echo "===================================="
echo "Cleaning up..."
echo "===================================="
docker-compose -f docker-compose.test.yml down