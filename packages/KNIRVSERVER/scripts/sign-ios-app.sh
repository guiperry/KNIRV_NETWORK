#!/bin/bash

# 1. Configure your Anisette Server URL
export ANISETTE_URL="http://localhost:6969"

# 2. Define your paths and credentials
PRIVATE_KEY=".config/anisette/private_key.p12"
KEY_PASSWORD="our_p12_password"
PROVISION_PROFILE=".config/anisette/free_profile.mobileprovision"

UNSIGNED_IPA="build/knirvconroller-ios-unsigned.ipa"
OUTPUT_IPA="build/knirvconroller-ios-release.ipa"

# 3. Verify files exist before signing
if [ ! -f "$PRIVATE_KEY" ] || [ ! -f "$PROVISION_PROFILE" ] || [ ! -f "$UNSIGNED_IPA" ]; then
    echo "❌ Error: One or more required files are missing. Check your paths!"
    exit 1
fi

echo "🚀 Exporting Anisette URL: $ANISETTE_URL"
echo "✍️  Signing $UNSIGNED_IPA with zsign..."

# 4. Execute zsign command
zsign -k "$PRIVATE_KEY" \
      -p "$KEY_PASSWORD" \
      -m "$PROVISION_PROFILE" \
      -o "$OUTPUT_IPA" \
      "$UNSIGNED_IPA"

# 5. Check if the signing was successful
if [ $? -eq 0 ]; then
    echo "✅ Success! Your signed IPA is ready at: $OUTPUT_IPA"
else
    echo "❌ Signing failed. Check the zsign output logs above."
fi
