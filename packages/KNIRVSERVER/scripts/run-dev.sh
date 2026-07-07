#!/usr/bin/env bash
set -e
# Set env vars directly for sudo — sudo passes all vars explicitly set here
cd "$(dirname "$0")"
exec sudo KNIRV_AUTH_REQUIRED=false KNIRV_ENVIRONMENT=development ./dist/knirv-server "$@"
