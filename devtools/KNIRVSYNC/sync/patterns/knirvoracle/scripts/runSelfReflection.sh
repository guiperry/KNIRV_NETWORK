#!/bin/bash



# Create a separate database directory for the Reflection
mkdir -p database_reflection

# Start a second reflection node on a different port with a different database path
echo "Starting test reflection node on port 5001..."
go run . -miners_address=testReflection65166fcb6516cb -port=5001 -database_path=database_reflection/agent_reflection.db --reflect http://127.0.0.1:5000 &

# Exit with the same status as the go command
exit $?