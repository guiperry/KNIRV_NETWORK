#!/bin/bash
cd "$(dirname "$0")/../agentic-wallet/go-backend"
echo "Starting KNIRVENGINE Agentic Wallet Server..."
go run cmd/server/main.go
