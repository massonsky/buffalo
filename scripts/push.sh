#!/usr/bin/env bash
set -euo pipefail

usage() { echo "Usage: $0 -m <commit message> [-maj <major>] [-min <minor>]"; exit 1; }

MESSAGE=""
MAJOR=1
MINOR=32

while [[ $# -gt 0 ]]; do
    case $1 in
        -m)   MESSAGE="$2"; shift 2 ;;
        -maj) MAJOR="$2"; shift 2 ;;
        -min) MINOR="$2"; shift 2 ;;
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

HASH=$(git rev-parse --short HEAD)
TAG="v${MAJOR}.${MINOR}.$HASH"

echo ""
echo "=== git tag $TAG ==="
git tag "$TAG"

echo ""
echo "=== git push ==="
git push -u origin HEAD
git push origin "$TAG"

echo ""
echo "Done: $TAG"
