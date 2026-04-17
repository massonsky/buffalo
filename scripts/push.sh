#!/usr/bin/env bash
set -euo pipefail

usage() { echo "Usage: $0 -m <commit message>"; exit 1; }

MESSAGE=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -m)   MESSAGE="$2"; shift 2 ;;
        *)    usage ;;
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

echo ""
echo "=== git push ==="
git push -u origin HEAD

echo ""
echo "Done (tag will be created by CI after successful build)"
