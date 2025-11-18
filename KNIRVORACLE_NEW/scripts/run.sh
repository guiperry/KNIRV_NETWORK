#!/bin/bash

# Check if the Go application is already running
if pgrep -f "go run" > /dev/null; then
    echo "Blockchain application is already running."
    exit 1
fi


# Run the Go application with the specified arguments
go run . -port 5000 -miners_address https://localhost:5000/rootChain -database_path database/agent.db
