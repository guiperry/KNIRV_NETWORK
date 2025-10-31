#!/bin/bash

# Fix import paths in Go files

echo "Fixing import paths..."

# Replace KNIRVNEXUS/backend with backend_server
find backend -name "*.go" -exec sed -i 's|KNIRVNEXUS/backend|backend_server|g' {} \;

# Replace other problematic imports
find backend -name "*.go" -exec sed -i 's|KNIRVORACLE/messages|backend_server/internal/messages|g' {} \;
find backend -name "*.go" -exec sed -i 's|KNIRVENGINE/desktop-client/database|backend_server/internal/database|g' {} \;

echo "Import paths fixed!"
