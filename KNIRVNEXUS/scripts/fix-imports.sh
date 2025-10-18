#!/bin/bash

# Fix import paths in Go files

echo "Fixing import paths..."

# Replace KNIRVNEXUS/backend with backend-server
find backend -name "*.go" -exec sed -i 's|KNIRVNEXUS/backend|backend-server|g' {} \;

# Replace other problematic imports
find backend -name "*.go" -exec sed -i 's|KNIRVORACLE/messages|backend-server/internal/messages|g' {} \;
find backend -name "*.go" -exec sed -i 's|KNIRVENGINE/desktop-client/database|backend-server/internal/database|g' {} \;

echo "Import paths fixed!"
