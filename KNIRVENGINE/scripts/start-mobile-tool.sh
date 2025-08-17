#!/bin/bash
cd "$(dirname "$0")/../mobile-tool"
echo "Starting KNIRVENGINE Mobile Tool development server..."
python3 -m http.server 8080
