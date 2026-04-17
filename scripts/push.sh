#!/usr/bin/env bash
set -euo pipefail

usage() { echo "Usage: $0 -m <commit message> [-maj <major>] [-min <minor>] [-pat <patch>]"; exit 1; }

MESSAGE=""
MAJOR=1
MINOR=32
PATCH=-1

while [[ $# -gt 0 ]]; do
    case $1 in
        -m)   MESSAGE="$2"; shift 2 ;;
        -maj) MAJOR="$2"; shift 2 ;;
        -min) MINOR="$2"; shift 2 ;;
        -pat) PATCH="$2"; shift 2 ;;
        *)    usage ;;
    esac
done
[ -z "$MESSAGE" ] && usage

# Auto-increment patch from latest tag if not specified
if [ "$PATCH" -lt 0 ]; then
    PREFIX="v${MAJOR}.${MINOR}."
    LATEST=$(git tag -l "${PREFIX}*" --sort=-v:refname 2>/dev/null | head -1)
    if [[ "$LATEST" =~ ^v[0-9]+\.[0-9]+\.([0-9]+)$ ]]; then
        PATCH=$(( ${BASH_REMATCH[1]} + 1 ))
    else
        PATCH=0
    fi
fi

echo "=== git add -A ==="
git add -A

echo ""
echo "=== git status ==="
git status

echo ""
echo "=== git commit ==="
git commit -m "$MESSAGE"

TAG="v${MAJOR}.${MINOR}.${PATCH}"

echo ""
echo "=== git tag $TAG ==="
git tag "$TAG"

echo ""
echo "=== git push ==="
git push -u origin HEAD
git push origin "$TAG"

echo ""
echo "Done: $TAG"
