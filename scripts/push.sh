#!/usr/bin/env bash
set -euo pipefail

usage() { echo "Usage: $0 -m <commit message>"; exit 1; }

MESSAGE=""
while getopts "m:" opt; do
    case $opt in
        m) MESSAGE="$OPTARG" ;;
        *) usage ;;
    esac
done
[ -z "$MESSAGE" ] && usage

echo "=== git add -A ==="
git add -A

echo ""
echo "=== git status ==="
git status

echo ""
echo "=== git commit ==="
git commit -m "$MESSAGE"

HASH=$(git rev-parse --short HEAD)
TAG="v1.32.$HASH"

echo ""
echo "=== git tag $TAG ==="
git tag "$TAG"

echo ""
echo "=== git push ==="
git push
git push --tags

echo ""
echo "Done: $TAG"
