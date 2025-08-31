#!/bin/bash

# Script to fix HTML footers and convert them to Markdown format for Docsify

# New Markdown footer format
NEW_FOOTER='
---

**Links:** [Code of Conduct](#/legal/CODE_OF_CONDUCT) | [Privacy Policy](#/legal/PRIVACY_POLICY) | [Terms and Conditions](#/legal/TERMS_AND_CONDITIONS)

© 2025 KNIRV Network'

# Find all README files with HTML footers and replace them
find KNIRVGATEWAY/documentation/docsify -name "README.md" -type f | while read file; do
    if grep -q "footer-links" "$file"; then
        echo "Fixing footer in $file"
        
        # Remove the HTML footer section
        sed -i '/<div class="footer-links">/,/<\/div>/d' "$file"
        
        # Add the new Markdown footer
        echo "$NEW_FOOTER" >> "$file"
    fi
done

echo "Footer fixes complete!"
