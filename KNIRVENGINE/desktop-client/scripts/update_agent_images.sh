#!/bin/bash

# Script to update all existing agents to use the new Agentify logo as default image
# This script compiles and runs the Go utility to update agent images

set -e

echo "🔄 Updating all existing agents to use the new Agentify logo..."
echo "=================================================="

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: This script must be run from the root directory of the KNIRV-Engine project"
    exit 1
fi

# Check if the update utility exists
if [ ! -d "scripts/update-agent-images" ]; then
    echo "❌ Error: Update utility not found at scripts/update-agent-images/"
    exit 1
fi

# Compile the update utility
echo "🔨 Compiling agent image update utility..."
go build -o scripts/update_agent_images ./scripts/update-agent-images

if [ $? -ne 0 ]; then
    echo "❌ Failed to compile update utility"
    exit 1
fi

echo "✅ Compilation successful"

# Run the update utility
echo ""
echo "🚀 Running agent image update..."
echo "================================"

./scripts/update_agent_images

# Clean up the compiled binary
echo ""
echo "🧹 Cleaning up..."
rm -f scripts/update_agent_images

echo ""
echo "🎉 Agent image update process completed!"
echo ""
echo "💡 Next steps:"
echo "   1. Restart the KNIRV-Engine application"
echo "   2. Refresh your browser to see the updated agent images"
echo "   3. All existing agents will now display the Agentify logo by default"
echo ""
