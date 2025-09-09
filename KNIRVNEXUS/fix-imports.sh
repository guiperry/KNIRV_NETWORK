#!/bin/bash

# Fix import paths in Go files

echo "Fixing import paths..."

# Replace KNIRVNEXUS/backend with nexus-backend
find backend -name "*.go" -exec sed -i 's|KNIRVNEXUS/backend|nexus-backend|g' {} \;

# Replace other problematic imports
find backend -name "*.go" -exec sed -i 's|KNIRVORACLE/messages|nexus-backend/internal/messages|g' {} \;
find backend -name "*.go" -exec sed -i 's|KNIRVENGINE/desktop-client/database|nexus-backend/internal/database|g' {} \;

echo "Import paths fixed!"
