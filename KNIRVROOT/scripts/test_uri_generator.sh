#!/bin/bash

# Test script for /uriGenerator endpoint and DHT functionality
SERVER_URL="http://localhost:5000"  # Default port - adjust if needed
URI_ENDPOINT="/uriGenerator"
INFO_ENDPOINT="/info"
PEERS_ENDPOINT="/devs"

echo "Testing KNIRVROOT Decentralized Discovery..."

# Test server info endpoint
echo "Checking server info..."
info_response=$(curl -s -X GET "${SERVER_URL}${INFO_ENDPOINT}")

# Check if curl succeeded
if [ $? -ne 0 ]; then
    echo "Error: Failed to connect to server for info"
    exit 1
fi

# Parse server info
http_port=$(echo "$info_response" | jq -r '.http_port')
p2p_port=$(echo "$info_response" | jq -r '.p2p_port')
chain_id=$(echo "$info_response" | jq -r '.chain_id')
dev_id=$(echo "$info_response" | jq -r '.dev_id')
multiaddrs=$(echo "$info_response" | jq -r '.multiaddrs')

echo "Server Info:"
echo "  HTTP Port: $http_port"
echo "  P2P Port: $p2p_port"
echo "  Chain ID: $chain_id"
echo "  Peer ID: $dev_id"
echo "  Multiaddresses: $multiaddrs"

# Test URI Generator endpoint with different scenarios
echo "Testing URI Generator endpoint..."

# Test with empty JSON body (should generate random UUID)
echo "Testing with empty JSON body..."
uri_response=$(curl -s -X POST "${SERVER_URL}${URI_ENDPOINT}" \
  -H "Content-Type: application/json" \
  -d '{}')

# Check if curl succeeded
if [ $? -ne 0 ]; then
    echo "Error: Failed to connect to server for URI generation"
    exit 1
fi

# Parse JSON response
uri=$(echo "$uri_response" | jq -r '.uri')
txn_hash=$(echo "$uri_response" | jq -r '.txn_hash')

# Verify response format
if [ -z "$uri" ] || [ -z "$txn_hash" ]; then
    echo "Error: Invalid response format"
    echo "Response: $uri_response"
    exit 1
fi

# Verify URI format
if [[ ! "$uri" =~ ^agent:// ]]; then
    echo "Error: Invalid URI format, expected agent:// prefix"
    echo "URI: $uri"
    exit 1
fi

# Extract components from URI
id=$(echo "$uri" | sed -n 's/^agent:\/\/\([^.]*\)\..*/\1/p')
resource_type=$(echo "$uri" | sed -n 's/^agent:\/\/[^.]*\.\([^\/]*\).*/\1/p')

echo "URI Components:"
echo "  ID: $id"
echo "  Resource Type: $resource_type"

# Test with desired ID
test_id="test-id-$(date +%s)"
echo "Testing with desired ID: $test_id..."
uri_response=$(curl -s -X POST "${SERVER_URL}${URI_ENDPOINT}" \
  -H "Content-Type: application/json" \
  -d "{\"desired_id\": \"$test_id\"}")

if [ $? -ne 0 ]; then
    echo "Error: Failed to request with desired ID"
    exit 1
fi

uri=$(echo "$uri_response" | jq -r '.uri')
if [[ ! "$uri" =~ $test_id ]]; then
    echo "Error: URI does not contain requested ID"
    echo "URI: $uri"
    exit 1
fi
echo "Successfully generated URI with desired ID"

# Test conflict scenario
echo "Testing conflict scenario..."
conflict_response=$(curl -s -X POST "${SERVER_URL}${URI_ENDPOINT}" \
  -H "Content-Type: application/json" \
  -d "{\"desired_id\": \"$test_id\"}")

if [ $? -ne 0 ]; then
    echo "Error: Failed to test conflict scenario"
    exit 1
fi

status_code=$(echo "$conflict_response" | jq -r '.status')
if [ "$status_code" != "409" ]; then
    echo "Error: Expected conflict (409) but got: $status_code"
    echo "Response: $conflict_response"
    exit 1
fi
echo "Successfully detected conflict for duplicate ID"

# Test devs endpoint
echo "Testing devs discovery..."
devs_response=$(curl -s -X GET "${SERVER_URL}${PEERS_ENDPOINT}")

# Check if curl succeeded
if [ $? -ne 0 ]; then
    echo "Error: Failed to connect to server for devs"
    exit 1
fi

# Parse devs response
devs_count=$(echo "$devs_response" | jq '. | length')
echo "Found $devs_count devs"

if [ "$devs_count" -gt 0 ]; then
    echo "Peers:"
    echo "$devs_response" | jq -r '.[]'
fi

# Output results
echo "Test successful!"
echo "Generated URI: $uri"
echo "Transaction Hash: $txn_hash"

exit 0