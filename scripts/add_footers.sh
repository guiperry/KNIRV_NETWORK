#!/bin/bash

# Script to add footers to all documentation pages that don't have them

FOOTER='
<div class="footer-links">
<a href="#/legal/CODE_OF_CONDUCT" class="footer-link">Code of Conduct</a> | <a href="#/legal/PRIVACY_POLICY" class="footer-link">Privacy Policy</a> | <a href="#/legal/TERMS_AND_CONDITIONS" class="footer-link">Terms and Conditions</a>

© 2025 KNIRV Network
</div>'

# List of README files to process (excluding legal files which already have footers)
FILES=(
    "KNIRVGATEWAY/documentation/docsify/knirvchain/README.md"
    "KNIRVGATEWAY/documentation/docsify/knirvcontroller/README.md"
    "KNIRVGATEWAY/documentation/docsify/knirvgateway/README.md"
    "KNIRVGATEWAY/documentation/docsify/knirvgraph/README.md"
    "KNIRVGATEWAY/documentation/docsify/knirvserver/README.md"
    "KNIRVGATEWAY/documentation/docsify/knirvoracle/README.md"
    "KNIRVGATEWAY/documentation/docsify/knirvrouter/README.md"
    "KNIRVGATEWAY/documentation/docsify/knirvsdk/README.md"
    "KNIRVGATEWAY/documentation/docsify/knirvtestnet/README.md"
    "KNIRVGATEWAY/documentation/docsify/knirvwallet/README.md"
    "KNIRVGATEWAY/documentation/docsify/whitepapers/README.md"
)

# Add footer to each file if it doesn't already have one
for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        # Check if footer already exists
        if ! grep -q "footer-links" "$file"; then
            echo "Adding footer to $file"
            echo "$FOOTER" >> "$file"
        else
            echo "Footer already exists in $file"
        fi
    else
        echo "File not found: $file"
    fi
done

echo "Footer addition complete!"
