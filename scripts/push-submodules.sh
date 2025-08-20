#!/bin/bash
#
# This script commits and pushes changes in all submodules,
# then commits and pushes the updated submodule references in the parent repo.

set -e

COMMIT_MSG="$1"

if [ -z "$COMMIT_MSG" ]; then
  echo -e "\033[0;31mError: Commit message is required.\033[0m"
  echo "Usage: ./scripts/push-submodules.sh \"Your commit message\""
  exit 1
fi

echo "--> Committing and pushing changes in all submodules with message: '$COMMIT_MSG'"
git submodule foreach '(git add . && (git diff-index --quiet HEAD || git commit -m "'"$COMMIT_MSG"'")) && git push || true'

echo "--> Staging and committing submodule updates in the main repo..."
git add .
if ! git diff-index --quiet HEAD; then
    git commit -m "chore: Update submodules to latest"
    echo "--> Pushing main repo..."
    git push
fi

echo "✅ All changes pushed successfully."